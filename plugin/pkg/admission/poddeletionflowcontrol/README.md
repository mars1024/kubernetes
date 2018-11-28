## 概述

sigma集群需要增加“删除pod流控”功能，一期参考2.0集群现行方案，采用了多级流控设计。

## 用法

支持用户制定精确到应用级别（映射到sigma 3.1中的namespace）的删除控制，缺省只提供全局的删除限流。

用户自定义限流策略，需要在应用namespace下创建名为pod-deletion-flow-control的configmap，形如：

```
apiVersion: v1
data:
  rules: '[{"duration":"1m","deleteLimit":1000},{"duration":"5m","deleteLimit":3000},{"duration":"1h","deleteLimit":6000},{"duration":"24h","deleteLimit":12000}]'
kind: ConfigMap
metadata:
  name: pod-deletion-flow-control
  namespace: app-namespace
```

针对规则的细节，这里做一些补充说明：

* 限流间隔的最小粒度是1分钟，最大粒度是1天
* 支持的时间单位是m和h
* 如果输入了1m50s，会向下取整作为1m处理

## 规则变更

如果因需要调整限流规则，直接修改对应namespace的configmap即可，生效时间为1分钟

## 白名单机制

白名单是在configmap中添加whitelist，是一组以逗号分割的用户名列表，用户名是用户通过各种认证方式（证书等）认证后的用户名，该用户的删除不计数。目前的主要应用场景是：k8s体系中，一次正常的删除一般是controller删除一次，kubelet删除一次，本功能主要是用于不统计kubelet的删除。示例：

```
apiVersion: v1
data:
  rules: '[{"duration":"1m","deleteLimit":1000},{"duration":"5m","deleteLimit":3000},{"duration":"1h","deleteLimit":6000},{"duration":"24h","deleteLimit":12000}]'
  whitelist: 'slave,whatever'
kind: ConfigMap
metadata:
  name: pod-deletion-flow-control
  namespace: app-namespace
```

## 补充说明

pod流控是工作于apiserver admission controller的，当前，sigma集群里apiserver工作于负载均衡模式，每个apiserver单独计数，所以，实际生效的限额应该是[配置限额数，配置限额数*apiserver实例数]