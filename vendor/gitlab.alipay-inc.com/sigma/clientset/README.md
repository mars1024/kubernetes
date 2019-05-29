# clientset

sigma 自定义的 CRD 资源定义和 client 操作库，和原生的 clientset 接口和使用方式保持一致。

要添加自己的 CRD types 定义，只需要两个步骤:

1. 在 `pkg/apis` 文件夹创建正确的 `/<group>/<version>/` 子目录，并编辑三个文件：
   - `doc.go`：package doc 文件，里面可以配置生成文件的 tags（见下面说明）
   - `types.go`：把自定义的 CRD spec 写到 `types.go` 文件中
2. 运行 `make` 生成对应的 client/informer/lister（每次修改文件都需要重新执行 make 命令）

在上面两个文件中有一些可以控制生成逻辑的 tags，如下：

- `doc.go`: package 级别的 tags
  - `// +k8s:deepcopy-gen=package`
    - [**Required**] Generating deepcopy methods for types.
  - `// +groupName=release.alipay-inc.com`
    - [**Required**] Used in the fake client as the full group name.
  - `// +k8s:defaulter-gen=TypeMeta`
    - [Optional] Generating default methods for types (`func SetDefaults_Release(obj *Release)`).
  - `// +k8s:conversion-gen=gitlab.alipay-inc.com/sigma/clientset/pkg/apis/release`
    - [Optional] Generating convension methods for types (`func Convert_v1alpha1_Release_To_release_Release(in *Release, out *release.Release, s conversion.Scope) error`).
- `types.go`: 单个 Spec 级别，放在 CRD 定义的注释上方
  - `// +genclient`
    - [**Required**] Generateing clients for types
  - `// +genclient:noStatus`
    - [**Required**] Generateing clients for types without method `UpdateStatus`.
  - `// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object`
    - [**Required**] Generating deepcopy methods with implementing `runtime.Object`.
  - `// +genclient:onlyVerbs=create,get`
    - [Optional] Generateing clients with specific verbs.
  - `// +genclient:skipVerbs=watch`
    - [Optional] Generateing clients without specific verbs.
  - `// +genclient:nonNamespaced`
    - [Optional] Generating global types rather than namespaced types.
  - `// +genclient:method=Scale,verb=update,subresource=scale,input=k8s.io/api/extensions/v1beta1.Scale,result=k8s.io/api/extensions/v1beta1.Scale`
    - [Optional] Generating external method.
