default: help
help:
\t@echo "just run: format, test, up, down"
format:
\tpre-commit run --all-files
up:
\tdocker compose up -d
down:
\tdocker compose down
