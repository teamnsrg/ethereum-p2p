# Ethnodes - MySQL 5.7 Node Database
- Based on the Docker Official Image packaging for MySQL Community Server 5.7
- Source: https://github.com/docker-library/mysql/tree/7b6d186052e268079972b4ea8c871f89161a899e/5.7
## Required software:
- Docker
- Docker Compose
## Usage:
1. Configure `.env`
    1. `MYSQL_USERNAME`
    2. `MYSQL_PASSWORD`
    3. `MYSQL_DB`
    4. `MYSQL_DIR`
    5. `BACKUP_DIR`
2. Run: `docker-compose up -d`
