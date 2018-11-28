
## configurations for alipodlifecyclehook

### ConfigMap

Namespace：kube-system

Name：sigma-alipodlifecyclehook-config


| key      | default value     | comment     |
| ---------- | :-----------:  | :-----------: |
| aone-image-name-contains     | docker.alibaba-inc.com/aone/     |     |
| volume-sigmalogs-name     | vol-sigmalogs     |     |
| volume-sigmalogs-path     | /var/log/sigma     |     |
| postStart-command     | for i in $(seq 1 60); do [ -x /home/admin/.start ] && break ; sleep 5 ; done; sudo -u admin /home/admin/.start>/var/log/sigma/start.log 2>&1 && sudo -u admin /home/admin/health.sh>>/var/log/sigma/start.log 2>&1     |     |
| preStop-command     | sudo -u admin /home/admin/stop.sh>/var/log/sigma/stop.log 2>&1     |     |
| probe-disable     | false     |     |
| probe-command     | sudo -u admin /home/admin/health.sh>/var/log/sigma/health.log 2>&1     |     |
| probe-timeout-seconds     | 20     |     |
| probe-period-seconds     | 60     |     |
| probe-period-seconds-specified     | {}     |  可以配置应用级别的probe周期，比如：{"buy2":1800}，即buy2应用probe周期为1800s   |

### Secret

Namespace：kube-system

Labels:
- usage: ali-registry-user-account
- username: aone (registry账号名)

Data:
- password: xxx (registry账号密码，base64编码)

如：
```$xslt
apiVersion: v1
kind: Secret
metadata:
  name: registry-user-aone
  namespace: kube-system
  labels:
    usage: ali-registry-user-account
    username: aone
data:
  password: xxx==
```

