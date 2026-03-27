# SMS DevOps Gateway

Gateway service nhận alert từ Alertmanager/VictoriaMetrics và forward thành tin nhắn SMS đến các receiver được cấu hình.

## 📋 Tổng quan

SMS DevOps Gateway là một service Go nhẹ, đóng vai trò trung gian giữa hệ thống monitoring (Alertmanager, VictoriaMetrics) và hệ thống gửi SMS. Service nhận webhook alert, xử lý và format message phù hợp, sau đó forward đến các số điện thoại đã cấu hình.

## ✨ Tính năng chính

### 1. **Nhận và xử lý Alert**
- Hỗ trợ format alert từ **Alertmanager** (Kubernetes)
- Hỗ trợ format alert từ **VictoriaMetrics** (VM)
- Parse và validate alert data tự động

### 2. **Routing thông minh**
- Route alert đến receiver dựa trên field `receiver` trong alert
- Fallback về `default_receiver` nếu không match
- Hỗ trợ gửi đến nhiều số điện thoại cho mỗi receiver

### 3. **Format message linh hoạt**
Service tự động format message dựa trên loại alert:

#### **Alert Kubernetes (K8s)**
```
[firing] d1-corep/infra-monitoring | pod-name | Pod is crash looping.
```
Bao gồm: cluster, namespace, pod, summary

#### **Alert Instance (VM/Server)**
```
[firing] AlertName: HostOutOfMemory | Instance: 10.68.40.199 | Sum: Host out of memory
```
Bao gồm: alertname, instance IP, summary

#### **Alert Message Queue (Kafka)**
```
[resolved] consumer_group_message_lag | ConsumerGroup: DATALAKE_0 | Job: dc1-kafka-core | Topic: FUND_DATALAKE | Sum: this is message queue
```
Bao gồm: alertname, consumer group, job, topic, summary

#### **Alert mặc định**
```
[firing] AlertGroup: vmagent | AlertName: PersistentQueueIsDroppingData | Sum: This is k8s alert
```

### 4. **Rule filtering**
Chỉ forward alert thỏa mãn:
- Status = `resolved` HOẶC
- Status = `firing` VÀ severity = `critical`

### 5. **Logging chi tiết**
- Log tất cả request nhận được
- Log full alert JSON
- Log message đã build
- Log receiver đã gửi
- Ghi vào file `/log/alerts.log` và console

### 6. **Multi-environment support**
- Timezone: UTC+7 (Asia/Ho_Chi_Minh)
- SSL certificates được cài sẵn
- Hỗ trợ chạy trên Docker và Kubernetes

## 🏗️ Kiến trúc

```
   ┌─────────────────┐
   │  Alertmanager   │
   │  VictoriaMetrics│
   └────────┬────────┘
            │ HTTP POST
            ▼
┌─────────────────────────┐
│  SMS DevOps Gateway     │
│  :8080/sms              │
│                         │
│  ┌──────────────────┐   │
│  │  Dispatcher      │   │
│  │  (handler)       │   │
│  └────────┬─────────┘   │
│           │             │
│  ┌────────▼─────────┐   │
│  │  Config Matcher  │   │
│  │  (routing)       │   │
│  └────────┬─────────┘   │
│           │             │
│  ┌────────▼─────────┐   │
│  │  Message Builder │   │
│  └────────┬─────────┘   │
│           │             │
│  ┌────────▼─────────┐   │
│  │  SMS Forwarder   │   │
│  └──────────────────┘   │
└───────────┬─────────────┘
            │ HTTP POST
            ▼
    ┌───────────────┐
    │  SMS Gateway  │
    │  (external)   │
    └───────────────┘
```

## 📁 Cấu trúc thư mục

```
sms-devops-gateway/
├── cmd/
│   └── main.go              # Entry point
├── config/
│   └── config.go            # Load và parse config
├── handler/
│   ├── dispatcher.go        # Route request
│   ├── handler.go           # Xử lý logic chính
│   ├── types.go             # Data structures
│   └── utils.go             # Helper functions
├── forwarder/
│   └── forwarder.go         # Forward SMS
├── k8s/
│   └── values.k8s-tool.dc1.yaml  # K8s manifests
├── config.json              # Cấu hình receiver
├── Dockerfile               # Multi-stage build
├── docker-compose.yml       # Local deployment
├── go.mod                   # Go dependencies
└── README.md
```

## ⚙️ Cấu hình

### File `config.json`

```json
{
  "receiver": [
    {
      "name": "alert-ops",
      "mobile": "0901234567, 0912345678"
    },
    {
      "name": "alert-devops",
      "mobile": "0923456789, 0934567890"
    },
    {
      "name": "alert-infra",
      "mobile": "0945678901"
    },
    {
      "name": "alert-d1-lgc-devops",
      "mobile": "0956789012, 0967890123"
    }
  ],
  "default_receiver": {
    "mobile": "0978901234"
  }
}
```

**Giải thích:**
- `receiver`: Danh sách các receiver, mỗi receiver có tên và danh sách số điện thoại (phân cách bởi dấu phẩy)
- `default_receiver`: Receiver mặc định khi không match được receiver nào

## 🚀 Triển khai

### 1. Chạy với Docker Compose (Local)

```bash
# Build và chạy
docker-compose up -d

# Xem logs
docker-compose logs -f

# Dừng service
docker-compose down
```

Service sẽ chạy tại `http://localhost:8080`

### 2. Chạy với Docker

```bash
# Build image
docker build -t sms-devops-gateway:latest .

# Chạy container
docker run -d \
  --name sms-gateway \
  -p 8080:8080 \
  -v $(pwd)/config.json:/config.json \
  sms-devops-gateway:latest
```

### 3. Triển khai trên Kubernetes

#### Bước 1: Chuẩn bị

```bash
# Tạo namespace
kubectl create namespace sms-devops-gateway

# Tạo secret cho Docker registry (nếu cần)
kubectl create secret docker-registry sms-devops-gateway-secret \
  --docker-server=your-registry.com \
  --docker-username=your-username \
  --docker-password=your-password \
  --docker-email=your-email \
  -n sms-devops-gateway
```

#### Bước 2: Update ConfigMap

Chỉnh sửa file `k8s/values.k8s-tool.dc1.yaml`, cập nhật phần `config.json` trong ConfigMap:

```yaml
data:
  config.json: |
    {
      "receiver": [
        {
          "name": "alert-devops",
          "mobile": "0901234567, 0912345678"
        }
      ],
      "default_receiver": {
        "mobile": "0978901234"
      }
    }
```

#### Bước 3: Deploy

```bash
# Apply tất cả resources
kubectl apply -f k8s/values.k8s-tool.dc1.yaml

# Kiểm tra deployment
kubectl get pods -n sms-devops-gateway
kubectl get svc -n sms-devops-gateway
kubectl get ingress -n sms-devops-gateway
```

#### Bước 4: Cấu hình Ingress (nếu cần)

Update domain trong Ingress:
```yaml
spec:
  rules:
    - host: sms-gateway.your-domain.com
```

### 4. Cấu hình Alertmanager

Thêm webhook vào Alertmanager config:

```yaml
receivers:
  - name: 'alert-devops'
    webhook_configs:
      - url: 'http://sms-gateway.sms-devops-gateway.svc.cluster.local/sms'
        send_resolved: true
```

Hoặc nếu dùng Ingress:
```yaml
receivers:
  - name: 'alert-devops'
    webhook_configs:
      - url: 'https://sms-gateway.your-domain.com/sms'
        send_resolved: true
```

## 🧪 Test thử

### Test với curl (K8s alert)

```bash
curl -X POST http://localhost:8080/sms \
  -H "Content-Type: application/json" \
  -d @alert-with-put.json
```

### Test với curl (VM alert - Instance)

```bash
curl -X POST http://localhost:8080/sms \
  -H "Content-Type: application/json" \
  -d @alert-with-put-vm-instance.json
```

### Test với curl (VM alert - Message Queue)

```bash
curl -X POST http://localhost:8080/sms \
  -H "Content-Type: application/json" \
  -d @alert-with-put-vm-message-queue.json
```

### Test response mẫu

**Success:**
```
HTTP/1.1 200 OK
Alert processed ✅
```

**Invalid format:**
```
HTTP/1.1 400 Bad Request
invalid alert format
```

**Alert ignored (không thỏa rule):**
```
HTTP/1.1 200 OK
⚠️ Alert ignored by default rules
```

## 📊 Monitoring

### Xem logs

**Docker:**
```bash
docker logs -f sms-gateway
```

**Kubernetes:**
```bash
kubectl logs -f -n sms-devops-gateway deployment/sms-gateway
```

**File log trong container:**
```bash
# Docker
docker exec sms-gateway tail -f /log/alerts.log

# Kubernetes
kubectl exec -n sms-devops-gateway deployment/sms-gateway -- tail -f /log/alerts.log
```

### Log format

```
[2025-10-16T10:30:00+07:00] Received alert:
{...full alert JSON...}

📥 Full Alert Received:
{...formatted JSON...}

📤 Built message: [firing] d1-corep/infra-monitoring | pod-name | Pod is crash looping.

📲 Message sent to receiver: alert-devops
```

## 🔧 Cấu hình nâng cao

### Custom SMS URL

Sửa file `forwarder/forwarder.go`:

```go
const smsURL = "http://your-sms-gateway:8082/sms/sendNumber"
```

Sau đó rebuild image:
```bash
docker build -t sms-devops-gateway:latest .
```

### Thay đổi timezone

Sửa Dockerfile:
```dockerfile
ENV TZ=Asia/Bangkok
```

### Thay đổi port

**Docker:**
```bash
docker run -p 9090:8080 sms-devops-gateway:latest
```

**Kubernetes:**
Sửa Service trong `values.k8s-tool.dc1.yaml`:
```yaml
ports:
  - port: 9090
    targetPort: 8080
```

### Tăng số replicas

Sửa Deployment trong `values.k8s-tool.dc1.yaml`:
```yaml
spec:
  replicas: 3
```

## 🐛 Troubleshooting

### Alert không được gửi

1. **Kiểm tra logs để xem alert có đến service không:**
```bash
kubectl logs -n sms-devops-gateway deployment/sms-gateway | grep "Received alert"
```

2. **Kiểm tra receiver name có match với config không:**
```bash
kubectl exec -n sms-devops-gateway deployment/sms-gateway -- cat /config.json
```

3. **Kiểm tra alert có thỏa mãn rule không:**
- Alert phải có `status = "resolved"` HOẶC `status = "firing"` với `severity = "critical"`

### SMS không được gửi đi

1. **Kiểm tra SMS Gateway URL:**
```bash
kubectl exec -n sms-devops-gateway deployment/sms-gateway -- cat /usr/bin/sms-devops-gateway
```

2. **Kiểm tra connectivity đến SMS Gateway:**
```bash
kubectl exec -n sms-devops-gateway deployment/sms-gateway -- wget -O- http://your-sms-gateway:8082/health
```

3. **Xem logs response từ SMS Gateway:**
```bash
kubectl logs -n sms-devops-gateway deployment/sms-gateway | grep "SMS sent"
```

### Container không start

1. **Kiểm tra config.json có hợp lệ không:**
```bash
cat config.json | jq .
```

2. **Kiểm tra volume mount:**
```bash
kubectl describe pod -n sms-devops-gateway | grep -A 5 Volumes
```

3. **Xem logs lỗi:**
```bash
kubectl logs -n sms-devops-gateway deployment/sms-gateway
```

### Pod CrashLoopBackOff

```bash
# Kiểm tra events
kubectl get events -n sms-devops-gateway --sort-by='.lastTimestamp'

# Kiểm tra describe pod
kubectl describe pod -n sms-devops-gateway <pod-name>

# Xem logs trước khi crash
kubectl logs -n sms-devops-gateway <pod-name> --previous
```

## 📝 Format Alert đầy đủ

### Alert từ Alertmanager (K8s)

```json
{
  "status": "firing",
  "alerts": [
    {
      "status": "firing",
      "labels": {
        "alertgroup": "kubernetes-apps",
        "alertname": "KubePodCrashLooping",
        "cluster": "d1-corep",
        "namespace": "infra-monitoring",
        "pod": "pod-name",
        "severity": "critical"
      },
      "annotations": {
        "summary": "Pod is crash looping.",
        "description": "Pod has been crash looping for 5 minutes"
      },
      "startsAt": "2025-06-10T16:32:30.000+07:00",
      "endsAt": "0001-01-01T00:00:00Z"
    }
  ]
}
```

### Alert từ VictoriaMetrics (Instance)

```json
{
  "receiver": "alert-d1-lgc-devops",
  "status": "firing",
  "alerts": [
    {
      "status": "firing",
      "labels": {
        "alertgroup": "Monitor Kafka Core",
        "alertname": "HostOutOfMemory",
        "instance": "10.9.8.7",
        "job": "dc1-kafka-core",
        "severity": "critical"
      },
      "annotations": {
        "summary": "Host out of memory",
        "description": "Memory usage above 90%"
      }
    }
  ]
}
```

### Alert từ VictoriaMetrics (Message Queue)

```json
{
  "receiver": "alert-d1-lgc-devops",
  "status": "firing",
  "alerts": [
    {
      "status": "firing",
      "labels": {
        "alertgroup": "Monitor Kafka Core",
        "alertname": "consumer_group_message_lag",
        "consumergroup": "DATALAKE_0",
        "job": "dc1-kafka-core",
        "topic": "FUND_DATALAKE",
        "severity": "critical"
      },
      "annotations": {
        "summary": "Consumer lag is high",
        "description": "Consumer group lag > 1000 messages"
      }
    }
  ]
}
```

## 🔐 Security Best Practices

1. **Sử dụng Secret cho SMS Gateway credentials:**
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: sms-gateway-creds
type: Opaque
stringData:
  api-key: your-api-key
```

2. **Giới hạn network policies:**
```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: sms-gateway-policy
spec:
  podSelector:
    matchLabels:
      app: sms-gateway
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: infra-monitoring
```

3. **Sử dụng RBAC:**
```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: sms-gateway-sa
  namespace: sms-devops-gateway
```

## 📈 Performance Tips

1. **Tăng replicas cho high availability:**
```yaml
replicas: 3
```

2. **Cấu hình resource limits:**
```yaml
resources:
  requests:
    memory: "64Mi"
    cpu: "100m"
  limits:
    memory: "128Mi"
    cpu: "200m"
```

3. **Cấu hình HPA (Horizontal Pod Autoscaler):**
```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: sms-gateway-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: sms-gateway
  minReplicas: 2
  maxReplicas: 5
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

## 📚 API Documentation

### Endpoint: `/sms`

**Method:** POST

**Content-Type:** application/json

**Request Body:**
- Alert format từ Alertmanager hoặc VictoriaMetrics

**Response:**
- `200 OK`: Alert được xử lý thành công
- `400 Bad Request`: Format alert không hợp lệ
- `500 Internal Server Error`: Lỗi server

**Example Request:**
```bash
curl -X POST http://localhost:8080/sms \
  -H "Content-Type: application/json" \
  -d '{
    "receiver": "alert-devops",
    "status": "firing",
    "alerts": [{
      "status": "firing",
      "labels": {
        "severity": "critical",
        "alertname": "HighCPU"
      },
      "annotations": {
        "summary": "CPU usage is high"
      }
    }]
  }'
```

## 🎯 Roadmap / Future Improvements

- [ ] Hỗ trợ file `ignore-alert.json` để ignore alert theo time window
- [ ] Thêm metrics endpoint (`/metrics`) cho Prometheus
- [ ] Hỗ trợ template message có thể customize
- [ ] Thêm retry mechanism với exponential backoff khi gửi SMS fail
- [ ] Rate limiting để tránh spam (max X SMS/phút)
- [ ] Web UI để quản lý config real-time
- [ ] Support multiple SMS providers
- [ ] Alert grouping và deduplication
- [ ] Health check endpoint (`/health`, `/ready`)
- [ ] Support cho Telegram, Slack notification

## 📞 Support

- **Issues:** Tạo issue trên repository
- **Email:** phamtung4030@gmail.com

## 📄 License

Internal use only - Proprietary

## 👥 Author
- PHAM THANH TUNG

---

**Version:** 1.0.0  
**Last Updated:** October 2025  
**Go Version:** 1.21+
