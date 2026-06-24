package skills

import (
	runtime "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/skills"
	"gorm.io/gorm"
)

const TemplateSkillID = "powerxplugin.template.basic"

func TemplateManifest() runtime.PluginSkillManifest {
	pkg, err := LoadTemplatePackage()
	if err != nil {
		panic(err)
	}
	return pkg.Manifest
}

func NewTemplateRegistry(db *gorm.DB) (*runtime.Registry, error) {
	reg := runtime.NewRegistry()
	reg.MustRegisterManifest(TemplateManifest())
	executor, err := NewTemplateSkillExecutor(db)
	if err != nil {
		return nil, err
	}
	if err := reg.RegisterExecutor(TemplateSkillID, executor.Execute); err != nil {
		return nil, err
	}
	return reg, nil
}

func MustTemplateRegistry(db *gorm.DB) *runtime.Registry {
	reg, err := NewTemplateRegistry(db)
	if err != nil {
		panic(err)
	}
	return reg
}

func NewTemplateRegistryHTTPAdapter(db *gorm.DB) *runtime.HTTPAdapter {
	return runtime.NewHTTPAdapter(MustTemplateRegistry(db))
}
