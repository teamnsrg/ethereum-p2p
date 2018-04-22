## Required software:
- Docker

## Usage:
1. Configure `.env`
    1. `ROOT_DIR`
    2. `ARCHIVE_DIR`
2. Run: `sudo ./build-images.sh`
2. Run: `sudo ./run.sh num_instance`

## Examples: schedule runs using `at` (as root):
1. `echo './run.sh 1' | at -m 7:00 pm`
2. `echo './tx-sniper-cron.sh 1' | at -m 7:01 pm`
