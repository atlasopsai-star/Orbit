# Orbit performance baseline

Measurements were collected on an Apple M4 (`darwin/arm64`) with:

```bash
go test ./src/pkg/orbitfs -run '^$' -bench 'BenchmarkSearchStream(1K|10K|100K)$' -benchmem -benchtime=1x
```

The benchmark names are retained for compatibility with the existing suite. They exercise the bounded filename search API and consume the result slice; they are not a terminal rendering benchmark.

| Fixture | Shape | Time/op | Bytes/op | Allocs/op |
| --- | --- | ---: | ---: | ---: |
| 1,000 entries | flat | 1.240 ms | 247,072 | 3,043 |
| 10,000 entries | flat | 10.213 ms | 2,532,880 | 29,258 |
| 100,000 entries | 1,000 directories | 3.409 ms | 677,480 | 8,722 |

The 100,000-entry case is intentionally bounded by Orbit's result cap, so its lower wall-clock time than the 10,000-entry case reflects early completion after enough matches are collected—not a claim that the entire tree was scanned. It is useful as a regression check for bounded memory and result handling, but not as a total traversal comparison.

## Interpretation

- Search remains bounded and does not allocate one renderable result for every matching path.
- The existing 1,000- and 10,000-entry measurements are the comparable full-result cases.
- A future performance pass should add separate instrumentation for time-to-first-batch and total traversal, especially for content search and folder-size scans.
- No automatic recursive folder-size calculation is performed during directory navigation.
