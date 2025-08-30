package platform

import (
	"fmt"
)

// AdapterFactory 平台适配器工厂
type AdapterFactory struct {
	platformConfigDir string
}

// NewAdapterFactory 创建平台适配器工厂
func NewAdapterFactory(platformConfigDir string) *AdapterFactory {
	return &AdapterFactory{
		platformConfigDir: platformConfigDir,
	}
}

// CreateAdapter 创建指定平台的适配器
func (f *AdapterFactory) CreateAdapter(platformType PlatformType) (PlatformAdapter, error) {
	switch platformType {
	case Linux:
		return NewLinuxAdapter(f.platformConfigDir), nil
	case Darwin:
		return NewDarwinAdapter(f.platformConfigDir), nil
	case IOS:
		return NewIOSAdapter(f.platformConfigDir), nil
	case Windows:
		return NewWindowsAdapter(f.platformConfigDir), nil
	default:
		return nil, fmt.Errorf("unsupported platform: %s", platformType)
	}
}

// GetDefaultAdapter 获取默认适配器（Linux）
func (f *AdapterFactory) GetDefaultAdapter() (PlatformAdapter, error) {
	return f.CreateAdapter(Linux)
}
