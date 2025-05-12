package e2e

//+kubebuilder:rbac:groups=etcd.fleet.agoda.com,resources=etcdclusters;etcdtenants,verbs=get;create;patch;delete
//+kubebuilder:rbac:groups=etcd.fleet.agoda.com,resources=etcdclusters/scale,verbs=get;update

//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list
//+kubebuilder:rbac:groups=core,resources=pods/eviction,verbs=create
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get
//+kubebuilder:rbac:groups=batch,resources=cronjobs,verbs=get
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;create
