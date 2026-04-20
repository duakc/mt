package gosys

import "github.com/duakc/mt/gosys/internal"

const GOOS = internal.GOOS

const IsAndroid = internal.IsAndroid == 1

const IsDarwin = internal.IsDarwin == 1 || internal.IsIos == 1

const IsDragonfly = internal.IsDragonfly == 1

const IsFreebsd = internal.IsFreebsd == 1

const IsHurd = internal.IsHurd == 1

const IsIllumos = internal.IsIllumos == 1

const IsIos = internal.IsIos == 1

const IsJs = internal.IsJs == 1

const IsLinux = internal.IsLinux == 1 || internal.IsAndroid == 1

const IsNacl = internal.IsNacl == 1

const IsNetbsd = internal.IsNetbsd == 1

const IsOpenbsd = internal.IsOpenbsd == 1

const IsPlan9 = internal.IsPlan9 == 1

const IsSolaris = internal.IsSolaris == 1

const IsWindows = internal.IsWindows == 1

const IsZos = internal.IsZos == 1
