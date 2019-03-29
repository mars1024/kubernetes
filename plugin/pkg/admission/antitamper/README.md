# Anti-Tamper Admission Controller

本Admission Controller的作用是防止**最终用户**篡改关键的ConfigMap或Labels、Annotations，对系统的可用性/稳定性造成影响。对上帝证书（我们自己的系统）是放行的，不会做任何保护。

最终这个功能应该以RBAC等更系统的方式来实现，所以这个Admission Controller只是一个临时措施，当有更好的技术可替代它时就会下线。


## 功能1: 保护某些资源不被修改

设置`(kind, namespace, name)`的白名单，来保护这些资源，用户不能修改任何field，也不能删除。


## 功能2: 保护所有资源的某些Label和Annotation不被修改

设置Label和Annotation的`key`，一旦资源创建完毕，防止用户修改任何资源中这些`key`所对应的Label或Annotation。

注：普通用户虽不能修改特定的Label或Annotation（包括不能从无到有地设置它们），但**可以删掉整个resource**，也可以在创建resource的时候指定它们的值（重点在于：不能Update）。


## 功能3: 保护某个Namespace下的所有资源不被修改

允许设置系统保留Namespace，系统保留Namespace下所有的资源都不能由最终用户进行修改。