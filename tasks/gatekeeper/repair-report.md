# Gatekeeper Task Repair Report

Generated: 2026-01-27 23:43:51

## Summary

| Status | Count |
|--------|-------|
| Repaired | 40 |
| No Changes | 24 |
| Errors | 0 |

---

## Repaired Tasks

### automount-serviceaccount-token

**File:** `../../tasks/gatekeeper/automount-serviceaccount-token/artifacts/beta-01.yaml`

```diff
--- /tmp/diff-a-78346908	2026-01-27 23:38:18.688907901 +0000
+++ /tmp/diff-b-3128938615	2026-01-27 23:38:18.688907901 +0000
@@ -11,3 +11,10 @@
   containers:
   - image: nginx
     name: nginx
+    resources:
+      limits:
+        cpu: 1m
+        memory: 1Mi
+      requests:
+        cpu: 1m
+        memory: 1Mi
\ No newline at end of file
```

### container-cpu-requests-memory-limits-and-requests

**File:** `../../tasks/gatekeeper/container-cpu-requests-memory-limits-and-requests/artifacts/alpha-01.yaml`

```diff
--- /tmp/diff-a-2200226529	2026-01-27 23:38:27.121702371 +0000
+++ /tmp/diff-b-3472015192	2026-01-27 23:38:27.121702371 +0000
@@ -16,8 +16,7 @@
     name: opa
     resources:
       limits:
-        cpu: 100m
-        memory: 1Gi
+        memory: 1Mi
       requests:
-        cpu: 100m
-        memory: 1Gi
+        cpu: 1m
+        memory: 1Mi
\ No newline at end of file
```

### container-cpu-requests-memory-limits-and-requests

**File:** `../../tasks/gatekeeper/container-cpu-requests-memory-limits-and-requests/artifacts/alpha-02.yaml`

```diff
--- /tmp/diff-a-570788602	2026-01-27 23:38:29.857960159 +0000
+++ /tmp/diff-b-3438119848	2026-01-27 23:38:29.857960159 +0000
@@ -16,7 +16,7 @@
     name: opa
     resources:
       limits:
-        memory: 2Gi
+        memory: 1Mi
       requests:
-        cpu: 100m
-        memory: 2Gi
+        cpu: 1m
+        memory: 1Mi
\ No newline at end of file
```

### container-cpu-requests-memory-limits-and-requests

**File:** `../../tasks/gatekeeper/container-cpu-requests-memory-limits-and-requests/artifacts/beta-01.yaml`

```diff
--- /tmp/diff-a-516922747	2026-01-27 23:38:35.422484400 +0000
+++ /tmp/diff-b-334066044	2026-01-27 23:38:35.422484400 +0000
@@ -16,5 +16,5 @@
     name: opa
     resources:
       requests:
-        cpu: 100m
-        memory: 2Gi
+        cpu: 1m
+        memory: 1Mi
\ No newline at end of file
```

### container-cpu-requests-memory-limits-and-requests

**File:** `../../tasks/gatekeeper/container-cpu-requests-memory-limits-and-requests/artifacts/beta-02.yaml`

```diff
--- /tmp/diff-a-2374235870	2026-01-27 23:38:44.727361024 +0000
+++ /tmp/diff-b-3330447585	2026-01-27 23:38:44.727361024 +0000
@@ -16,4 +16,4 @@
     name: opa
     resources:
       limits:
-        memory: 2Gi
+        memory: 1Mi
\ No newline at end of file
```

### container-image-must-have-digest

**File:** `../../tasks/gatekeeper/container-image-must-have-digest/artifacts/alpha-01.yaml`

```diff
--- /tmp/diff-a-170031435	2026-01-27 23:39:03.157097301 +0000
+++ /tmp/diff-b-3308686937	2026-01-27 23:39:03.157097301 +0000
@@ -13,3 +13,10 @@
     - --addr=localhost:8080
     image: openpolicyagent/opa:0.9.2@sha256:04ff8fce2afd1a3bc26260348e5b290e8d945b1fad4b4c16d22834c2f3a1814a
     name: opa
+    resources:
+      limits:
+        cpu: 1m
+        memory: 1Mi
+      requests:
+        cpu: 1m
+        memory: 1Mi
\ No newline at end of file
```

### container-image-must-have-digest

**File:** `../../tasks/gatekeeper/container-image-must-have-digest/artifacts/beta-01.yaml`

```diff
--- /tmp/diff-a-562702146	2026-01-27 23:39:11.421875926 +0000
+++ /tmp/diff-b-2531433285	2026-01-27 23:39:11.421875926 +0000
@@ -13,6 +13,13 @@
     - --addr=localhost:8080
     image: openpolicyagent/opa:0.9.2
     name: opa
+    resources:
+      limits:
+        cpu: 1m
+        memory: 1Mi
+      requests:
+        cpu: 1m
+        memory: 1Mi
   initContainers:
   - command:
     - opa
@@ -20,3 +27,10 @@
     - "true"
     image: openpolicyagent/opa:0.9.2
     name: opainit
+    resources:
+      limits:
+        cpu: 1m
+        memory: 1Mi
+      requests:
+        cpu: 1m
+        memory: 1Mi
\ No newline at end of file
```

### container-limits

**File:** `../../tasks/gatekeeper/container-limits/artifacts/alpha-01.yaml`

```diff
--- /tmp/diff-a-2127529261	2026-01-27 23:39:15.430253554 +0000
+++ /tmp/diff-b-189592540	2026-01-27 23:39:15.430253554 +0000
@@ -16,5 +16,5 @@
     name: opa
     resources:
       limits:
-        cpu: 100m
-        memory: 1Gi
+        cpu: 1m
+        memory: 1Mi
\ No newline at end of file
```

### container-limits

**File:** `../../tasks/gatekeeper/container-limits/artifacts/beta-01.yaml`

```diff
--- /tmp/diff-a-2276698345	2026-01-27 23:39:22.002872757 +0000
+++ /tmp/diff-b-3007353064	2026-01-27 23:39:22.002872757 +0000
@@ -16,5 +16,5 @@
     name: opa
     resources:
       limits:
-        cpu: 100m
-        memory: 2Gi
+        cpu: 201m
+        memory: 1025Mi
\ No newline at end of file
```

### container-limits-and-requests

**File:** `../../tasks/gatekeeper/container-limits-and-requests/artifacts/alpha-01.yaml`

```diff
--- /tmp/diff-a-935213274	2026-01-27 23:39:26.283276012 +0000
+++ /tmp/diff-b-2451037990	2026-01-27 23:39:26.283276012 +0000
@@ -16,8 +16,8 @@
     name: opa
     resources:
       limits:
-        cpu: 100m
-        memory: 1Gi
+        cpu: 1m
+        memory: 1Mi
       requests:
-        cpu: 100m
-        memory: 1Gi
+        cpu: 1m
+        memory: 1Mi
\ No newline at end of file
```

### container-limits-and-requests

**File:** `../../tasks/gatekeeper/container-limits-and-requests/artifacts/beta-01.yaml`

```diff
--- /tmp/diff-a-2956983673	2026-01-27 23:39:33.879991692 +0000
+++ /tmp/diff-b-3721382430	2026-01-27 23:39:33.879991692 +0000
@@ -16,5 +16,5 @@
     name: opa
     resources:
       requests:
-        cpu: 100m
-        memory: 2Gi
+        cpu: 1m
+        memory: 1Mi
\ No newline at end of file
```

### container-limits-and-requests

**File:** `../../tasks/gatekeeper/container-limits-and-requests/artifacts/beta-02.yaml`

```diff
--- /tmp/diff-a-3278240938	2026-01-27 23:39:40.388604861 +0000
+++ /tmp/diff-b-2936149935	2026-01-27 23:39:40.388604861 +0000
@@ -16,6 +16,6 @@
     name: opa
     resources:
       limits:
-        memory: 2Gi
+        memory: 1Mi
       requests:
-        cpu: 100m
+        cpu: 1m
\ No newline at end of file
```

### container-limits-ignore-cpu

**File:** `../../tasks/gatekeeper/container-limits-ignore-cpu/artifacts/alpha-01.yaml`

```diff
--- /tmp/diff-a-547288287	2026-01-27 23:39:49.033419277 +0000
+++ /tmp/diff-b-397118715	2026-01-27 23:39:49.033419277 +0000
@@ -15,4 +15,5 @@
     name: opa
     resources:
       limits:
-        memory: 1Gi
+        cpu: 1m
+        memory: 1Mi
\ No newline at end of file
```

### container-limits-ignore-cpu

**File:** `../../tasks/gatekeeper/container-limits-ignore-cpu/artifacts/beta-01.yaml`

```diff
--- /tmp/diff-a-4136303550	2026-01-27 23:39:58.662326399 +0000
+++ /tmp/diff-b-3590720589	2026-01-27 23:39:58.662326399 +0000
@@ -15,4 +15,4 @@
     name: opa
     resources:
       limits:
-        memory: 2Gi
+        memory: 1025Mi
\ No newline at end of file
```

### container-requests

**File:** `../../tasks/gatekeeper/container-requests/artifacts/alpha-01.yaml`

```diff
--- /tmp/diff-a-2482243402	2026-01-27 23:40:01.578601133 +0000
+++ /tmp/diff-b-3947023390	2026-01-27 23:40:01.578601133 +0000
@@ -16,5 +16,5 @@
     name: opa
     resources:
       requests:
-        cpu: 100m
-        memory: 1Gi
+        cpu: 1m
+        memory: 1Mi
\ No newline at end of file
```

### container-requests

**File:** `../../tasks/gatekeeper/container-requests/artifacts/beta-01.yaml`

```diff
--- /tmp/diff-a-2676974274	2026-01-27 23:40:16.291987248 +0000
+++ /tmp/diff-b-2159002527	2026-01-27 23:40:16.291987248 +0000
@@ -16,5 +16,4 @@
     name: opa
     resources:
       requests:
-        cpu: 100m
-        memory: 2Gi
+        memory: 1Mi
\ No newline at end of file
```

### disallow-interactive

**File:** `../../tasks/gatekeeper/disallow-interactive/artifacts/beta-01.yaml`

```diff
--- /tmp/diff-a-303059914	2026-01-27 23:40:28.157105027 +0000
+++ /tmp/diff-b-4069340378	2026-01-27 23:40:28.157105027 +0000
@@ -10,5 +10,12 @@
   containers:
   - image: nginx
     name: nginx
+    resources:
+      limits:
+        cpu: 1m
+        memory: 1Mi
+      requests:
+        cpu: 1m
+        memory: 1Mi
     stdin: true
-    tty: true
+    tty: true
\ No newline at end of file
```

### disallowed-tags

**File:** `../../tasks/gatekeeper/disallowed-tags/artifacts/alpha-01.yaml`

```diff
--- /tmp/diff-a-2466849591	2026-01-27 23:40:32.461510532 +0000
+++ /tmp/diff-b-2373017443	2026-01-27 23:40:32.461510532 +0000
@@ -13,3 +13,10 @@
     - --addr=localhost:8080
     image: openpolicyagent/opa:0.9.2
     name: opa
+    resources:
+      limits:
+        cpu: 1m
+        memory: 1Mi
+      requests:
+        cpu: 1m
+        memory: 1Mi
\ No newline at end of file
```

### disallowed-tags

**File:** `../../tasks/gatekeeper/disallowed-tags/artifacts/alpha-02.yaml`

```diff
--- /tmp/diff-a-2704542440	2026-01-27 23:40:39.582181347 +0000
+++ /tmp/diff-b-2084005033	2026-01-27 23:40:39.582181347 +0000
@@ -13,15 +13,36 @@
     - --addr=localhost:8080
     image: openpolicyagent/opa-exp:latest
     name: opa-exp
+    resources:
+      limits:
+        cpu: 1m
+        memory: 1Mi
+      requests:
+        cpu: 1m
+        memory: 1Mi
   - args:
     - run
     - --server
     - --addr=localhost:8080
     image: openpolicyagent/init:v1
     name: opa-init
+    resources:
+      limits:
+        cpu: 1m
+        memory: 1Mi
+      requests:
+        cpu: 1m
+        memory: 1Mi
   - args:
     - run
     - --server
     - --addr=localhost:8080
     image: openpolicyagent/opa-exp2:latest
     name: opa-exp2
+    resources:
+      limits:
+        cpu: 1m
+        memory: 1Mi
+      requests:
+        cpu: 1m
+        memory: 1Mi
\ No newline at end of file
```

### disallowed-tags

**File:** `../../tasks/gatekeeper/disallowed-tags/artifacts/beta-01.yaml`

```diff
--- /tmp/diff-a-1908910629	2026-01-27 23:40:45.454734573 +0000
+++ /tmp/diff-b-18871720	2026-01-27 23:40:45.454734573 +0000
@@ -11,5 +11,12 @@
     - run
     - --server
     - --addr=localhost:8080
-    image: openpolicyagent/opa
+    image: openpolicyagent/opa:latest
     name: opa
+    resources:
+      limits:
+        cpu: 1m
+        memory: 1Mi
+      requests:
+        cpu: 1m
+        memory: 1Mi
\ No newline at end of file
```

### disallowed-tags

**File:** `../../tasks/gatekeeper/disallowed-tags/artifacts/beta-02.yaml`

```diff
--- /tmp/diff-a-2600767606	2026-01-27 23:40:51.643317576 +0000
+++ /tmp/diff-b-4042534248	2026-01-27 23:40:51.643317576 +0000
@@ -11,5 +11,12 @@
     - run
     - --server
     - --addr=localhost:8080
-    image: openpolicyagent:443/opa
+    image: openpolicyagent/opa:latest
     name: opa
+    resources:
+      limits:
+        cpu: 1m
+        memory: 1Mi
+      requests:
+        cpu: 1m
+        memory: 1Mi
\ No newline at end of file
```

### disallowed-tags

**File:** `../../tasks/gatekeeper/disallowed-tags/artifacts/beta-03.yaml`

```diff
--- /tmp/diff-a-2157204205	2026-01-27 23:40:58.647977457 +0000
+++ /tmp/diff-b-1885083131	2026-01-27 23:40:58.647977457 +0000
@@ -13,3 +13,10 @@
     - --addr=localhost:8080
     image: openpolicyagent/opa:latest
     name: opa
+    resources:
+      limits:
+        cpu: 1m
+        memory: 1Mi
+      requests:
+        cpu: 1m
+        memory: 1Mi
\ No newline at end of file
```

### disallowed-tags

**File:** `../../tasks/gatekeeper/disallowed-tags/artifacts/beta-04.yaml`

```diff
--- /tmp/diff-a-3467799881	2026-01-27 23:41:07.948853653 +0000
+++ /tmp/diff-b-40507317	2026-01-27 23:41:07.948853653 +0000
@@ -13,21 +13,49 @@
     - --addr=localhost:8080
     image: openpolicyagent/opa-exp:latest
     name: opa
+    resources:
+      limits:
+        cpu: 1m
+        memory: 1Mi
+      requests:
+        cpu: 1m
+        memory: 1Mi
   - args:
     - run
     - --server
     - --addr=localhost:8080
     image: openpolicyagent/init:latest
     name: opa-init
+    resources:
+      limits:
+        cpu: 1m
+        memory: 1Mi
+      requests:
+        cpu: 1m
+        memory: 1Mi
   - args:
     - run
     - --server
     - --addr=localhost:8080
     image: openpolicyagent/opa-exp2:latest
     name: opa-exp2
+    resources:
+      limits:
+        cpu: 1m
+        memory: 1Mi
+      requests:
+        cpu: 1m
+        memory: 1Mi
   - args:
     - run
     - --server
     - --addr=localhost:8080
     image: openpolicyagent/monitor:latest
     name: opa-monitor
+    resources:
+      limits:
+        cpu: 1m
+        memory: 1Mi
+      requests:
+        cpu: 1m
+        memory: 1Mi
\ No newline at end of file
```

### ephemeral-storage-limit

**File:** `../../tasks/gatekeeper/ephemeral-storage-limit/artifacts/alpha-01.yaml`

```diff
--- /tmp/diff-a-1531720932	2026-01-27 23:41:12.253259152 +0000
+++ /tmp/diff-b-2203059188	2026-01-27 23:41:12.253259152 +0000
@@ -16,6 +16,6 @@
     name: opa
     resources:
       limits:
-        cpu: 100m
-        ephemeral-storage: 100Mi
-        memory: 1Gi
+        cpu: 1m
+        ephemeral-storage: 1Mi
+        memory: 1Mi
\ No newline at end of file
```

### ephemeral-storage-limit

**File:** `../../tasks/gatekeeper/ephemeral-storage-limit/artifacts/alpha-02.yaml`

```diff
--- /tmp/diff-a-2849427152	2026-01-27 23:41:17.557758864 +0000
+++ /tmp/diff-b-1546856517	2026-01-27 23:41:17.557758864 +0000
@@ -16,9 +16,9 @@
     name: opa
     resources:
       limits:
-        cpu: 100m
-        ephemeral-storage: 100Mi
-        memory: 1Gi
+        cpu: 1m
+        ephemeral-storage: 1Mi
+        memory: 1Mi
   initContainers:
   - command:
     - opa
@@ -28,6 +28,6 @@
     name: init-opa
     resources:
       limits:
-        cpu: 100m
-        ephemeral-storage: 100Mi
-        memory: 1Gi
+        cpu: 1m
+        ephemeral-storage: 1Mi
+        memory: 1Mi
\ No newline at end of file
```

### ephemeral-storage-limit

**File:** `../../tasks/gatekeeper/ephemeral-storage-limit/artifacts/beta-01.yaml`

```diff
--- /tmp/diff-a-1592087040	2026-01-27 23:41:30.542982135 +0000
+++ /tmp/diff-b-3911189410	2026-01-27 23:41:30.542982135 +0000
@@ -16,5 +16,6 @@
     name: opa
     resources:
       limits:
-        cpu: 100m
-        memory: 2Gi
+        cpu: 1m
+        memory: 1Mi
+        ephemeral-storage: 501Mi
\ No newline at end of file
```

### ephemeral-storage-limit

**File:** `../../tasks/gatekeeper/ephemeral-storage-limit/artifacts/beta-02.yaml`

```diff
--- /tmp/diff-a-863495414	2026-01-27 23:41:37.475635224 +0000
+++ /tmp/diff-b-4277370106	2026-01-27 23:41:37.475635224 +0000
@@ -16,6 +16,6 @@
     name: opa
     resources:
       limits:
-        cpu: 100m
-        ephemeral-storage: 1Pi
-        memory: 1Gi
+        cpu: 1m
+        ephemeral-storage: 501Mi
+        memory: 1Mi
\ No newline at end of file
```

### ephemeral-storage-limit

**File:** `../../tasks/gatekeeper/ephemeral-storage-limit/artifacts/beta-03.yaml`

```diff
--- /tmp/diff-a-1531917066	2026-01-27 23:41:45.660406265 +0000
+++ /tmp/diff-b-793745700	2026-01-27 23:41:45.660406265 +0000
@@ -16,9 +16,9 @@
     name: opa
     resources:
       limits:
-        cpu: 100m
-        ephemeral-storage: 100Mi
-        memory: 1Gi
+        cpu: 1m
+        ephemeral-storage: 1Mi
+        memory: 1Mi
   initContainers:
   - command:
     - opa
@@ -28,6 +28,6 @@
     name: init-opa
     resources:
       limits:
-        cpu: 100m
-        ephemeral-storage: 1Pi
-        memory: 1Gi
+        cpu: 1m
+        ephemeral-storage: 501Mi
+        memory: 1Mi
\ No newline at end of file
```

### horizontal-pod-autoscaler

**File:** `../../tasks/gatekeeper/horizontal-pod-autoscaler/artifacts/beta-03.yaml`

```diff
--- /tmp/diff-a-2983499820	2026-01-27 23:42:17.319388662 +0000
+++ /tmp/diff-b-1994504033	2026-01-27 23:42:17.319388662 +0000
@@ -6,7 +6,7 @@
   name: resource-004
   namespace: gk-horizontal-pod-autoscaler
 spec:
-  maxReplicas: 6
+  maxReplicas: 3
   metrics:
   - resource:
       name: cpu
@@ -18,4 +18,4 @@
   scaleTargetRef:
     apiVersion: apps/v1
     kind: Deployment
-    name: nginx-deployment-missing
+    name: nginx-deployment-missing
\ No newline at end of file
```

### memory-and-cpu-ratios

**File:** `../../tasks/gatekeeper/memory-and-cpu-ratios/artifacts/alpha-01.yaml`

```diff
--- /tmp/diff-a-418598189	2026-01-27 23:42:25.236134446 +0000
+++ /tmp/diff-b-3824165010	2026-01-27 23:42:25.236134446 +0000
@@ -16,8 +16,8 @@
     name: opa
     resources:
       limits:
-        cpu: "4"
-        memory: 2Gi
+        cpu: 1m
+        memory: 1Mi
       requests:
-        cpu: "1"
-        memory: 2Gi
+        cpu: 1m
+        memory: 1Mi
\ No newline at end of file
```

### memory-and-cpu-ratios

**File:** `../../tasks/gatekeeper/memory-and-cpu-ratios/artifacts/beta-01.yaml`

```diff
--- /tmp/diff-a-2859673095	2026-01-27 23:42:33.060871560 +0000
+++ /tmp/diff-b-3186090629	2026-01-27 23:42:33.060871560 +0000
@@ -16,8 +16,8 @@
     name: opa
     resources:
       limits:
-        cpu: "4"
-        memory: 2Gi
+        cpu: 11m
+        memory: 1Mi
       requests:
-        cpu: 100m
-        memory: 2Gi
+        cpu: 1m
+        memory: 1Mi
\ No newline at end of file
```

### memory-ratio-only

**File:** `../../tasks/gatekeeper/memory-ratio-only/artifacts/alpha-01.yaml`

```diff
--- /tmp/diff-a-333234004	2026-01-27 23:42:38.493383319 +0000
+++ /tmp/diff-b-600699231	2026-01-27 23:42:38.493383319 +0000
@@ -16,8 +16,8 @@
     name: opa
     resources:
       limits:
-        cpu: 200m
-        memory: 200Mi
+        cpu: 1m
+        memory: 1Mi
       requests:
-        cpu: 100m
-        memory: 100Mi
+        cpu: 1m
+        memory: 1Mi
\ No newline at end of file
```

### memory-ratio-only

**File:** `../../tasks/gatekeeper/memory-ratio-only/artifacts/beta-01.yaml`

```diff
--- /tmp/diff-a-3834777727	2026-01-27 23:42:43.897892439 +0000
+++ /tmp/diff-b-4195284976	2026-01-27 23:42:43.897892439 +0000
@@ -16,8 +16,8 @@
     name: opa
     resources:
       limits:
-        cpu: 800m
-        memory: 2Gi
+        cpu: 3m
+        memory: 3Mi
       requests:
-        cpu: 100m
-        memory: 100Mi
+        cpu: 1m
+        memory: 1Mi
\ No newline at end of file
```

### must-have-key

**File:** `../../tasks/gatekeeper/must-have-key/artifacts/alpha-01.yaml`

```diff
--- /tmp/diff-a-2718704646	2026-01-27 23:42:50.554519502 +0000
+++ /tmp/diff-b-4021106310	2026-01-27 23:42:50.554519502 +0000
@@ -10,3 +10,12 @@
   containers:
   - image: nginx
     name: nginx
+    resources:
+      limits:
+        cpu: 1m
+        ephemeral-storage: 1Mi
+        memory: 1Mi
+      requests:
+        cpu: 1m
+        ephemeral-storage: 1Mi
+        memory: 1Mi
\ No newline at end of file
```

### must-have-key

**File:** `../../tasks/gatekeeper/must-have-key/artifacts/beta-01.yaml`

```diff
--- /tmp/diff-a-2940881907	2026-01-27 23:42:53.526799497 +0000
+++ /tmp/diff-b-3746930988	2026-01-27 23:42:53.526799497 +0000
@@ -10,3 +10,10 @@
   containers:
   - image: nginx
     name: nginx
+    resources:
+      limits:
+        cpu: 1m
+        memory: 1Mi
+      requests:
+        cpu: 1m
+        memory: 1Mi
\ No newline at end of file
```

### replica-limit

**File:** `../../tasks/gatekeeper/replica-limit/artifacts/beta-01.yaml`

```diff
--- /tmp/diff-a-971123652	2026-01-27 23:43:00.599465759 +0000
+++ /tmp/diff-b-1840081684	2026-01-27 23:43:00.599465759 +0000
@@ -20,3 +20,10 @@
         name: nginx
         ports:
         - containerPort: 80
+        resources:
+          limits:
+            cpu: 1m
+            memory: 1Mi
+          requests:
+            cpu: 1m
+            memory: 1Mi
\ No newline at end of file
```

### repo-must-not-be-k8s-gcr-io

**File:** `../../tasks/gatekeeper/repo-must-not-be-k8s-gcr-io/artifacts/beta-01.yaml`

```diff
--- /tmp/diff-a-1270850099	2026-01-27 23:43:11.740515270 +0000
+++ /tmp/diff-b-2733640009	2026-01-27 23:43:11.740515270 +0000
@@ -9,3 +9,10 @@
   containers:
   - image: k8s.gcr.io/kustomize/kustomize:v3.8.9
     name: kustomize
+    resources:
+      limits:
+        cpu: 1m
+        memory: 1Mi
+      requests:
+        cpu: 1m
+        memory: 1Mi
\ No newline at end of file
```

### repo-must-not-be-k8s-gcr-io

**File:** `../../tasks/gatekeeper/repo-must-not-be-k8s-gcr-io/artifacts/beta-02.yaml`

```diff
--- /tmp/diff-a-2201616429	2026-01-27 23:43:18.153119349 +0000
+++ /tmp/diff-b-820326659	2026-01-27 23:43:18.153119349 +0000
@@ -9,6 +9,20 @@
   containers:
   - image: registry.k8s.io/kustomize/kustomize:v3.8.9
     name: kustomize
+    resources:
+      limits:
+        cpu: 1m
+        memory: 1Mi
+      requests:
+        cpu: 1m
+        memory: 1Mi
   initContainers:
   - image: k8s.gcr.io/kustomize/kustomize:v3.8.9
     name: kustomizeinit
+    resources:
+      limits:
+        cpu: 1m
+        memory: 1Mi
+      requests:
+        cpu: 1m
+        memory: 1Mi
\ No newline at end of file
```

### repo-must-not-be-k8s-gcr-io

**File:** `../../tasks/gatekeeper/repo-must-not-be-k8s-gcr-io/artifacts/beta-03.yaml`

```diff
--- /tmp/diff-a-3606733424	2026-01-27 23:43:26.085866628 +0000
+++ /tmp/diff-b-1904620589	2026-01-27 23:43:26.085866628 +0000
@@ -9,6 +9,20 @@
   containers:
   - image: k8s.gcr.io/kustomize/kustomize:v3.8.9
     name: kustomize
+    resources:
+      limits:
+        cpu: 1m
+        memory: 1Mi
+      requests:
+        cpu: 1m
+        memory: 1Mi
   initContainers:
   - image: k8s.gcr.io/kustomize/kustomize:v3.8.9
     name: kustomizeinit
+    resources:
+      limits:
+        cpu: 1m
+        memory: 1Mi
+      requests:
+        cpu: 1m
+        memory: 1Mi
\ No newline at end of file
```

### required-probes

**File:** `../../tasks/gatekeeper/required-probes/artifacts/beta-02.yaml`

```diff
--- /tmp/diff-a-1271132577	2026-01-27 23:43:51.908299132 +0000
+++ /tmp/diff-b-2868361635	2026-01-27 23:43:51.908299132 +0000
@@ -17,6 +17,13 @@
     ports:
     - containerPort: 80
     readinessProbe: null
+    resources:
+      limits:
+        cpu: 1m
+        memory: 1Mi
+      requests:
+        cpu: 1m
+        memory: 1Mi
     volumeMounts:
     - mountPath: /tmp/cache
       name: cache-volume
@@ -29,6 +36,13 @@
       periodSeconds: 10
       tcpSocket:
         port: 8080
+    resources:
+      limits:
+        cpu: 1m
+        memory: 1Mi
+      requests:
+        cpu: 1m
+        memory: 1Mi
   volumes:
   - emptyDir: {}
-    name: cache-volume
+    name: cache-volume
\ No newline at end of file
```

---

## No Changes Needed

- allowed-repos
- allowed-repos
- allowed-repos
- allowed-repos
- allowed-reposv2
- allowed-reposv2
- allowed-reposv2
- allowed-reposv2
- allowed-reposv2
- allowed-reposv2
- allowed-reposv2
- automount-serviceaccount-token
- block-loadbalancer-services
- block-loadbalancer-services
- container-cpu-requests-memory-limits-and-requests
- container-limits-and-requests
- disallow-interactive
- horizontal-pod-autoscaler
- horizontal-pod-autoscaler
- horizontal-pod-autoscaler
- replica-limit
- repo-must-not-be-k8s-gcr-io
- required-probes
- required-probes

---

