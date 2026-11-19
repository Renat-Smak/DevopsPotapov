#!/usr/bin/env python3
import json
import math
import os
import re
import shutil
import statistics
import subprocess
import tempfile
import time
from pathlib import Path

ROOT = Path(__file__).resolve().parent
PREFIX = f"lr9-{os.environ.get('GITHUB_RUN_ID', 'local')}-{os.getpid()}"
CONTAINERS = set()
IMAGES = set()


def run(*args, timeout=45):
    result = subprocess.run(args, capture_output=True, text=True, timeout=timeout)
    if result.returncode:
        raise RuntimeError(f"{' '.join(args)}\n{result.stdout}{result.stderr}")
    return result.stdout.strip()


def docker(*args, timeout=45):
    return run("docker", *args, timeout=timeout)


def section(title):
    print(f"\n>>> {title}", flush=True)


def start(image, port=8080):
    cid = docker("run", "-d", "--label", f"devops.lab09={PREFIX}",
                 "-p", f"127.0.0.1::{port}", image, timeout=90)
    CONTAINERS.add(cid)
    host = docker("port", cid, f"{port}/tcp").splitlines()[0]
    return cid, f"http://{host}"


def remove(cid):
    docker("rm", "-fv", cid)
    CONTAINERS.remove(cid)


def get(url, path):
    return run("curl", "-fsS", "--max-time", "2", url + path)


def wait_http(url, path="/health", code="200", seconds=15):
    deadline = time.monotonic() + seconds
    while time.monotonic() < deadline:
        try:
            status = run("curl", "-s", "--max-time", "1", "-o", "/dev/null",
                         "-w", "%{http_code}", url + path, timeout=2)
            if status == code:
                return status
        except (RuntimeError, subprocess.TimeoutExpired):
            pass
        time.sleep(0.04)
    raise RuntimeError(f"{url}{path} did not return HTTP {code}")


def wait_health(cid, wanted):
    deadline = time.monotonic() + 30
    while time.monotonic() < deadline:
        actual = docker("inspect", "--format", "{{.State.Health.Status}}", cid)
        if actual == wanted:
            return actual
        time.sleep(0.5)
    debug = docker("inspect", "--format", "{{json .State.Health}}", cid)
    raise RuntimeError(f"expected {wanted}: {debug}")


def check_endpoints(url):
    for path, count in [("/count", 0), ("/", 1), ("/count", 1), ("/", 2), ("/count", 2)]:
        body = get(url, path)
        assert json.loads(body) == {"count": count}, (path, body)
        print(f"GET {path} -> {body}")


def build(image, path, *options):
    IMAGES.add(image)
    process = subprocess.run(
        ["docker", "build", "--progress=plain", "-t", image, *options, str(path)],
        capture_output=True, text=True, timeout=420,
    )
    log = process.stdout + "\n" + process.stderr
    if process.returncode:
        raise RuntimeError("\n".join(log.splitlines()[-75:]))
    return log


def context_line(log):
    step = re.search(r"^(#\d+) \[internal\] load build context", log, re.M)
    assert step, log
    lines = [line for line in log.splitlines()
             if line.startswith(f"{step[1]} transferring context:")]
    assert lines, log
    return lines[-1]


def image_size(image):
    return int(docker("image", "inspect", image, "--format", "{{.Size}}"))


def image_id(image):
    return docker("image", "inspect", image, "--format", "{{.Id}}")


def binary_size(image):
    return int(docker("run", "--rm", "--entrypoint", "stat", image,
                      "-c", "%s", "/usr/local/bin/counter"))


def timings(images):
    cold = {key: [] for key in images}
    warm = {key: [] for key in images}
    section("LATENCY")
    print("Cold: docker run to first HTTP 200; warm: curl GET /count time_total")
    for trial in range(5):
        keys = list(images) if trial % 2 == 0 else list(reversed(images))
        for key in keys:
            start_ns = time.perf_counter_ns()
            cid, url = start(images[key])
            try:
                wait_http(url, "/count")
                cold[key].append((time.perf_counter_ns() - start_ns) / 1e6)
                for _ in range(12):
                    value = run("curl", "-fsS", "--max-time", "2", "-o", "/dev/null",
                                "-w", "%{time_total}", f"{url}/count")
                    warm[key].append(float(value) * 1000)
            finally:
                remove(cid)
    for key in images:
        samples = sorted(warm[key])
        p95 = samples[math.ceil(0.95 * len(samples)) - 1]
        print(f"{key}: cold median={statistics.median(cold[key]):.2f} ms (n={len(cold[key])})")
        print(f"{key}: warm median={statistics.median(samples):.3f} ms; p95={p95:.3f} ms (n={len(samples)})")


def check_registry(source, aliases):
    section("LOOPBACK_REGISTRY")
    registry, url = start("registry:2", 5000)
    wait_http(url, "/v2/")
    remote = url.removeprefix("http://") + "/counter"
    expected_id = image_id(source)
    for tag in ("1.0.0", "latest"):
        target = f"{remote}:{tag}"
        IMAGES.add(target)
        docker("tag", source, target)
        output = docker("push", target, timeout=90)
        print(output.splitlines()[-1])
    docker("image", "rm", *aliases, f"{remote}:latest", f"{remote}:1.0.0")
    seen = subprocess.run(["docker", "image", "inspect", f"{remote}:1.0.0"],
                          capture_output=True, text=True, timeout=10)
    assert seen.returncode != 0, "registry tag should be absent before pull"
    print("Local tags removed; pulling 1.0.0 from loopback registry.")
    print(docker("pull", f"{remote}:1.0.0", timeout=90).splitlines()[-1])
    assert image_id(f"{remote}:1.0.0") == expected_id
    print(f"Pulled image ID matches: {expected_id}")
    cid, address = start(f"{remote}:1.0.0")
    wait_http(address)
    count = get(address, "/")
    assert json.loads(count) == {"count": 1}
    print(f"Pulled image GET / -> {count}")
    remove(cid)
    remove(registry)


def main():
    section("ENGINE")
    print(docker("version", "--format", "Docker client={{.Client.Version}} server={{.Server.Version}}"))
    naive, ignored = f"{PREFIX}:naive", f"{PREFIX}:naive-ignore"
    plain, packed, bad = f"{PREFIX}:plain", f"{PREFIX}:1.0.0", f"{PREFIX}:unhealthy"
    with tempfile.TemporaryDirectory(prefix="devops-lab09-") as temp:
        base = Path(temp) / "no-ignore"
        base.mkdir()
        for name in ("go.mod", "main.go", "main_test.go", "Dockerfile", "Dockerfile.naive", "README.md"):
            shutil.copyfile(ROOT / name, base / name)
        (base / "tmp").mkdir()
        for path in (base / "tmp" / "probe.bin", base / "sample.log"):
            path.write_bytes(os.urandom(1024 * 1024))
        filtered = Path(temp) / "with-ignore"
        shutil.copytree(base, filtered)
        shutil.copyfile(ROOT / ".dockerignore", filtered / ".dockerignore")
        section("NAIVE")
        log1 = build(naive, base, "-f", str(base / "Dockerfile.naive"))
        print(f"naive image bytes={image_size(naive)}")
        cid, url = start(naive)
        wait_http(url)
        check_endpoints(url)
        remove(cid)
        section("DOCKERIGNORE")
        log2 = build(ignored, filtered, "-f", str(filtered / "Dockerfile.naive"))
        print(f"without .dockerignore: {context_line(log1)}")
        print(f"with .dockerignore:    {context_line(log2)}")
        for image, expected in [(naive, True), (ignored, False)]:
            marker = docker("run", "--rm", "--entrypoint", "sh", image,
                            "-c", "if [ -f /src/sample.log ]; then echo present; else echo absent; fi")
            assert (marker == "present") == expected
            print(f"{image.split(':')[-1]} /src/sample.log: {marker}")

    section("MULTISTAGE_UPX")
    build(plain, ROOT, "--target", "plain")
    packed_log = build(packed, ROOT, "--target", "release")
    for line in packed_log.splitlines():
        if re.search(r"Ultimate Packer|\d+\.\d+%|\[OK\]", line):
            print(line)
    build(bad, ROOT, "--target", "unhealthy")
    for label, image in [("naive", naive), ("alpine plain", plain), ("alpine upx", packed)]:
        print(f"{label}: image bytes={image_size(image)} | binary bytes={binary_size(image)}")
    assert binary_size(packed) < binary_size(plain) and image_size(packed) < image_size(plain) < image_size(naive)

    section("TAGS")
    aliases = [packed]
    for tag in ("1.0", "1", "latest", "stable"):
        alias = f"{PREFIX}:{tag}"
        docker("tag", packed, alias)
        aliases.append(alias)
        IMAGES.add(alias)
    assert len({image_id(alias) for alias in aliases}) == 1
    print(docker("image", "ls", "--digests", "--no-trunc", PREFIX,
                 "--format", "table {{.Tag}}\t{{.Digest}}\t{{.ID}}"))

    section("HEALTH")
    for label, image, state, status in [("plain", plain, "healthy", "200"),
                                        ("upx", packed, "healthy", "200"),
                                        ("upx with forced error", bad, "unhealthy", "500")]:
        cid, url = start(image)
        wait_http(url, code=status)
        observed = wait_health(cid, state)
        print(f"{label}: GET /health -> HTTP {status}; Docker status={observed}")
        if image == packed:
            check_endpoints(url)
        remove(cid)

    timings({"plain": plain, "upx": packed})
    check_registry(packed, aliases)
    section("LAB09_OK")


if __name__ == "__main__":
    try:
        main()
    finally:
        for cid in CONTAINERS.copy():
            try:
                remove(cid)
            except RuntimeError as error:
                print(error)
        if IMAGES:
            subprocess.run(["docker", "image", "rm", *sorted(IMAGES)],
                           capture_output=True, timeout=60)
