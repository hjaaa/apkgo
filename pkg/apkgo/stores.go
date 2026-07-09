package apkgo

// Import built-in store packages so the embeddable apkgo package has the
// same registry contents as the CLI without requiring callers to blank import
// every store themselves.
import (
	_ "github.com/KevinGong2013/apkgo/v3/pkg/store/fir"
	_ "github.com/KevinGong2013/apkgo/v3/pkg/store/googleplay"
	_ "github.com/KevinGong2013/apkgo/v3/pkg/store/honor"
	_ "github.com/KevinGong2013/apkgo/v3/pkg/store/huawei"
	_ "github.com/KevinGong2013/apkgo/v3/pkg/store/oppo"
	_ "github.com/KevinGong2013/apkgo/v3/pkg/store/pgyer"
	_ "github.com/KevinGong2013/apkgo/v3/pkg/store/samsung"
	_ "github.com/KevinGong2013/apkgo/v3/pkg/store/script"
	_ "github.com/KevinGong2013/apkgo/v3/pkg/store/tencent"
	_ "github.com/KevinGong2013/apkgo/v3/pkg/store/vivo"
	_ "github.com/KevinGong2013/apkgo/v3/pkg/store/xiaomi"
)
