package rbac

// Permission 描述插件需要声明的权限键。
type Permission struct {
	Key   string `json:"key"`
	Scope string `json:"scope"`
	Desc  string `json:"desc,omitempty"`
}

// Reporter 定义框架或宿主用于接收权限声明的接口。
type Reporter interface {
	RegisterPermissions(perms []Permission) error
}

// Report 将权限声明上报给宿主实现。
func Report(target Reporter, perms []Permission) error {
	if len(perms) == 0 {
		return nil
	}
	if err := Validate(perms); err != nil {
		return err
	}
	if target == nil {
		return nil
	}
	return target.RegisterPermissions(perms)
}
