FROM python:3.12-slim

WORKDIR /app

# Install dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy application
COPY sync_server.py .

# Create data directory
RUN mkdir -p /app/data

# Environment variables
ENV HOST=0.0.0.0
ENV PORT=8080
ENV DATA_DIR=/app/data

EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD python -c "import urllib.request; urllib.request.urlopen('http://localhost:8080/health')" || exit 1

CMD ["python", "sync_server.py"]
