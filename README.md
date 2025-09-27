# Panasonic Prometheus Exporter

> A simple, standalone Prometheus exporter for Panasonic breaker box energy data.

This exporter fetches a CSV file containing energy data from a Panasonic breaker box URL, parses the values, and exposes them in a Prometheus-friendly format.

## Features

* **Standalone**: No external dependencies besides Go to compile.
* **Efficient**: Fetches data only when scraped by Prometheus.
* **Secure**: Configure the URL and circuit mappings via a `.env` file.
* **Robust**: Designed to be run as a `systemd` service.

## Installation

You must have a recent version of Go installed.

```bash
# Clone the repository
git clone https://github.com/styygeli/panasonic-exporter.git
cd panasonic-exporter

# Tidy dependencies
go mod tidy

# Build the optimized binary
go build -ldflags="-s -w"
```
This will create a `panasonic-exporter` executable in the directory.

## Configuration

The exporter is configured via a file named `.env` placed in the same directory as the executable.

1.  Create a file named `.env`:
    ```bash
    nano .env
    ```

2.  Add the URL for your breaker box and the JSON mappings for your circuits:
    ```ini
    # URL to the CSV data file from the breaker box
    PANASONIC_URL="http://192.168.1.100/csv/InstVal.csv"

    # JSON map of desired metrics to their column index in the CSV
    # The key is the 'entity' label, and the value is the column number.
    PANASONIC_MAPPINGS='{"main": 5, "ecocute": 6, "kitchen_appliances": 7, "living_room": 9}'
    ```

## Running the Exporter

### For Testing

You can run the exporter directly from your terminal.

```bash
./panasonic-exporter
```
The exporter will start on port `9190`. You can now test the metrics endpoint:
```bash
curl http://localhost:9190/metrics
```

### As a `systemd` Service

1.  Move the compiled binary and the `.env` file to a dedicated directory:
    ```bash
    # Replace 'your-user' with your actual username
    mkdir -p /home/your-user/panasonic-exporter
    mv panasonic-exporter .env /home/your-user/panasonic-exporter/
    ```

2.  Create a `systemd` service file at `/etc/systemd/system/panasonic-exporter.service`:
    ```ini
    [Unit]
    Description=Prometheus Exporter for Panasonic Breaker Box
    Wants=network-online.target
    After=network-online.target

    [Service]
    # Replace 'your-user' with your actual username
    User=your-user
    Group=your-user
    
    WorkingDirectory=/home/your-user/panasonic-exporter
    ExecStart=/home/your-user/panasonic-exporter/panasonic-exporter
    
    Restart=on-failure
    RestartSec=5s

    [Install]
    WantedBy=multi-user.target
    ```

3.  Enable and start the service:
    ```bash
    sudo systemctl daemon-reload
    sudo systemctl enable --now panasonic-exporter.service
    sudo systemctl status panasonic-exporter.service
    ```

## Exposed Metrics

The exporter exposes the following metrics:

| Metric                  | Labels                | Description                         |
| ----------------------- | --------------------- | ----------------------------------- |
| `panasonic_power_watts` | `entity`, `friendly_name` | Current power consumption in Watts. |

It also includes standard Go process and `promhttp` metrics for monitoring the exporter's own health.

## License

This project is licensed under the MIT License.
