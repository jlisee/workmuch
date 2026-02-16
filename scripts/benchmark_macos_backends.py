#!/usr/bin/env python3
import argparse
import statistics
import sys
import time
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from usageinfo_macos import UsageInfoMacOS


def _percentile(values, p):
    if not values:
        return 0.0
    if len(values) == 1:
        return values[0]
    ordered = sorted(values)
    idx = int(round((len(ordered) - 1) * p))
    return ordered[idx]


def benchmark_backend(backend, iterations):
    usage = UsageInfoMacOS(backend=backend)
    samples_ms = []
    try:
        for _ in range(iterations):
            start = time.perf_counter()
            usage.getUsageInfo()
            end = time.perf_counter()
            samples_ms.append((end - start) * 1000.0)
    finally:
        usage.release()

    avg_ms = statistics.fmean(samples_ms) if samples_ms else 0.0
    p50_ms = _percentile(samples_ms, 0.50)
    p95_ms = _percentile(samples_ms, 0.95)
    return {
        "avg_ms": avg_ms,
        "p50_ms": p50_ms,
        "p95_ms": p95_ms,
    }


def print_report(backend, result):
    print(f"\\nBackend: {backend}")
    print(f"  avg latency: {result['avg_ms']:.3f} ms")
    print(f"  p50 latency: {result['p50_ms']:.3f} ms")
    print(f"  p95 latency: {result['p95_ms']:.3f} ms")
    for hz in (1, 10, 30):
        cpu_pct = (result["avg_ms"] * hz) / 10.0
        print(f"  estimated 1-core CPU at {hz:>2}Hz: {cpu_pct:.2f}%")


def main():
    parser = argparse.ArgumentParser(
        description="Benchmark subprocess vs native macOS usage-info backends."
    )
    parser.add_argument(
        "--iterations",
        type=int,
        default=500,
        help="Number of samples to query per backend (default: 500)",
    )
    args = parser.parse_args()

    if sys.platform != "darwin":
        raise SystemExit("This benchmark only runs on macOS.")

    print(f"Running {args.iterations} samples per backend...")
    subprocess_result = benchmark_backend("subprocess", args.iterations)
    native_result = benchmark_backend("native", args.iterations)

    print_report("subprocess", subprocess_result)
    print_report("native", native_result)


if __name__ == "__main__":
    main()
