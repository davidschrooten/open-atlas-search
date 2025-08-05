# Kubernetes Health Check Endpoints

This document describes the health check endpoints designed for Kubernetes liveness and readiness probes.

## Endpoints Overview

### 1. `/health` - Liveness Probe
- **Purpose**: Simple health check to verify the application is alive
- **Use Case**: Kubernetes liveness probe to restart unhealthy pods
- **Response**: Always returns 200 OK if the service is running

**Response Format:**
```json
{
  "status": "healthy"
}
```

**Usage in Kubernetes:**
```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 30
  periodSeconds: 10
  timeoutSeconds: 5
  failureThreshold: 3
```

### 2. `/ready` - Readiness Probe
- **Purpose**: Comprehensive readiness check to verify the service can handle traffic
- **Use Case**: Kubernetes readiness probe to control traffic routing
- **Checks Performed**:
  - Search engine initialization
  - Indexer service initialization
  - Ability to list indexes (tests core functionality)
  - Presence of configured indexes (if any are configured)

**Success Response (200 OK):**
```json
{
  "status": "ready",
  "checks": {
    "searchEngine": "ok",
    "indexerService": "ok",
    "indexes": "ok"
  }
}
```

**Failure Response (503 Service Unavailable):**
- Returns appropriate error message as plain text
- Examples:
  - `"search engine not initialized"`
  - `"indexer service not initialized"`
  - `"search engine not ready"`
  - `"no indexes available"`

**Usage in Kubernetes:**
```yaml
readinessProbe:
  httpGet:
    path: /ready
    port: 8080
  initialDelaySeconds: 15
  periodSeconds: 5
  timeoutSeconds: 3
  failureThreshold: 3
  successThreshold: 1
```

## Complete Kubernetes Deployment Example

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: open-atlas-search
  labels:
    app: open-atlas-search
spec:
  replicas: 2
  selector:
    matchLabels:
      app: open-atlas-search
  template:
    metadata:
      labels:
        app: open-atlas-search
    spec:
      containers:
      - name: open-atlas-search
        image: open-atlas-search:latest
        ports:
        - containerPort: 8080
        env:
        - name: MONGO_URI
          valueFrom:
            secretKeyRef:
              name: mongodb-secret
              key: uri
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 15
          periodSeconds: 5
          timeoutSeconds: 3
          failureThreshold: 3
          successThreshold: 1
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        volumeMounts:
        - name: config
          mountPath: /etc/config
        - name: indexes
          mountPath: /var/lib/indexes
      volumes:
      - name: config
        configMap:
          name: open-atlas-search-config
      - name: indexes
        persistentVolumeClaim:
          claimName: open-atlas-search-indexes
---
apiVersion: v1
kind: Service
metadata:
  name: open-atlas-search-service
  labels:
    app: open-atlas-search
spec:
  selector:
    app: open-atlas-search
  ports:
  - protocol: TCP
    port: 8080
    targetPort: 8080
  type: ClusterIP
```

## Health Check Behavior

### Startup Sequence
1. **Container starts** - `/health` returns 200 OK immediately
2. **Services initialize** - `/ready` returns 503 until initialization complete
3. **Indexes load** - `/ready` verifies indexes are available
4. **Ready for traffic** - `/ready` returns 200 OK, pod receives traffic

### Failure Scenarios

#### Liveness Probe Failures
- Service completely unresponsive
- HTTP server crashed
- **Action**: Kubernetes restarts the pod

#### Readiness Probe Failures
- Search engine initialization failed
- Cannot connect to MongoDB
- No indexes available when expected
- **Action**: Kubernetes removes pod from service endpoints (no traffic)

## Monitoring and Alerting

You can use these endpoints for external monitoring:

```bash
# Quick health check
curl http://your-service:8080/health

# Detailed readiness check
curl http://your-service:8080/ready

# Comprehensive status (includes sync times)
curl http://your-service:8080/status
```

## Best Practices

1. **Different Thresholds**: Use different `failureThreshold` values for liveness vs readiness
2. **Appropriate Delays**: Set `initialDelaySeconds` based on your startup time
3. **Resource Limits**: Ensure adequate CPU/memory for health checks to respond
4. **Logging**: Health check failures are logged for debugging
5. **Monitoring**: Consider alerting on repeated readiness probe failures

## Troubleshooting

### Pod Keeps Restarting
- Check liveness probe configuration
- Verify `/health` endpoint is responding
- Review application logs for crashes

### Pod Not Receiving Traffic
- Check readiness probe status: `kubectl describe pod <pod-name>`
- Test `/ready` endpoint manually: `kubectl port-forward <pod-name> 8080:8080`
- Review MongoDB connectivity and index initialization

### Slow Startup
- Increase `initialDelaySeconds` for readiness probe
- Monitor index loading time
- Consider warming indexes in init containers
