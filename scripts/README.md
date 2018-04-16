# Ethnodes - MySQL 5.7 Node Database
- Based on the Docker Official Image packaging for MySQL Community Server 5.7
- Source: https://github.com/docker-library/mysql/tree/7b6d186052e268079972b4ea8c871f89161a899e/5.7
## Required software:
- Docker
- ~~Docker Compose~~
## Usage:
1. Configure `.env`
    1. `MYSQL_USERNAME`
    2. `MYSQL_PASSWORD`
    3. `MYSQL_HOST`
    4. `MYSQL_DB`
    5. `ROOT_DIR`
2. Run: `sudo ./build-images.sh`
2. Run: `sudo ./run.sh num_instance`

## Examples: schedule runs using `at` (as root):
1. `echo './run.sh 30' | at -m 7:00 pm`
2. `echo './logrotate-cron.sh' | at -m 7:01 pm`
