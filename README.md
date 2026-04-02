# Weight Tracker

A simple weight tracking app with a visualization showing daily weight and 7-day exponential moving average.

## Screenshot

![Weight Tracker Dashboard](screenshot.png)

## Features

- Record daily weight
- Weight history visualization
- Weight trend analysis with exponential moving average (EMA) smoothing
- CSV export and import
- Persistent SQLite database

## Running Locally

```bash
cd src
go run .
```

Visit `http://localhost:8080`

## Docker

```bash
docker build -t weight-tracker .
docker run -p 8080:8080 weight-tracker
```

## Kubernetes

I use these manifests to deploy to my home k3s cluster:

```bash
kubectl apply -f k8s/manifest.yaml
```

## Configuration

| Environment Variable | Default      | Description                  |
|----------------------|--------------|------------------------------|
| `DB_PATH`            | `weights.db` | Path to the SQLite database file |

## Database

Weight data is stored in a SQLite database (`weights.db` by default). Data can be exported or imported as CSV via the app UI.
