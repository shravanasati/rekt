# rekt

<p align="center">
 <img src="logo.png" width=225 height=200>
</p>

A [superfast](#benchmarks) cross platform port killer tool.

### Installation

rekt is available only on Linux and Windows as of now.

#### From releases (recommended)
Download the latest release for your platform from the [releases page](https://github.com/shravanasati/rekt/releases).

#### Using Go (requires Go>=1.24)
```bash
go install github.com/shravanasati/rekt@latest
```

### Usage

##### Find process occupying a port
```bash
rekt 8000
```

##### Terminate the process
```bash
rekt 8000 -t
```

##### Kill the process
```bash
rekt 8000 -k
```

You may want to run rekt with sudo permissions to find and kill processes started by users other than you.

Do note that on Linux, `--terminate/-t` sends SIGTERM and `--kill/-k` sends SIGKILL.
On Windows, the behavior of both flags is identical.

### Benchmarks

rekt is consistently 8x faster than fuser and 4x faster than lsof.

Here are results from a run on my PC.

| Command | Runs | Average [ms] | User [ms] | System [ms] | Min [ms] | Max [ms] | Relative |
| ------- | ---- | ------- | ---- | ------ | --- | --- | -------- |
`rekt 8000` | 690 | 3.92 ± 0.40 | 2.79 | 8.89 | 3.33 | 5.66 | 1.00 ± 0.00 
`lsof -i :8000` | 179 | 16.74 ± 3.23 | 3.58 | 12.18 | 14.42 | 38.14 | 4.27 ± 0.93 
`fuser 8000/tcp` | 97 | 32.00 ± 2.36 | 11.16 | 20.48 | 29.62 | 38.94 | 8.16 ± 1.03 


You can run these benchmarks yourself using [atomic](https://github.com/shravanasati/atomic) or [hyperfine](https://github.com/sharkdp/hyperfine) as follows:


1. Start a server in background.

	`python -m http.server 8000 &`

2. Run the benchmark.

	`atomic "fuser 8000/tcp" "lsof -i :8000" "rekt 8000" -w 10`

	OR

	`hyperfine "fuser 8000/tcp" "lsof -i :8000" "rekt 8000" -w 10 -N`