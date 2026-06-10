package main

import "time"

// GoCommand represents a native go toolchain command
type GoCommand string

const (
	GoCmdMod      GoCommand = "mod"
	GoCmdTidy     GoCommand = "tidy"
	GoCmdBuild    GoCommand = "build"
	GoCmdRun      GoCommand = "run"
	GoCmdInstall  GoCommand = "install"
	GoCmdFmt      GoCommand = "fmt"
	GoCmdVet      GoCommand = "vet"
	GoCmdTest     GoCommand = "test"
	GoCmdGet      GoCommand = "get"
	GoCmdList     GoCommand = "list"
	GoCmdDownload GoCommand = "download"
	GoCmdVerify   GoCommand = "verify"
	GoCmdInit     GoCommand = "init"
	GoCmdClean    GoCommand = "clean"
	GoCmdVersion  GoCommand = "version"
	GoCmdEnv      GoCommand = "env"
	GoCmdGenerate GoCommand = "generate"
)

// CraftPath represents standardized folder/file names within the .craft ecosystem
type CraftPath string

const (
	CraftHomeDir       CraftPath = ".craft"
	CraftToolchainsDir CraftPath = "toolchains"
	CraftCacheDir      CraftPath = "cache"
	CraftScriptCache   CraftPath = "script-cache"
	CraftContentHash   CraftPath = ".craft-content-hash"
)

const (
	CraftDocsURL            = "https://craft.trymagic.xyz/docs.html"
	RemoteVersionsCacheFile = "remote_versions.json"
	RemoteVersionsCacheTTL  = 24 * time.Hour
)

// TargetOS represents the operating system target for compilation or toolchain
type TargetOS string

const (
	OSWindows   TargetOS = "windows"
	OSLinux     TargetOS = "linux"
	OSDarwin    TargetOS = "darwin"
	OSFreeBSD   TargetOS = "freebsd"
	OSOpenBSD   TargetOS = "openbsd"
	OSNetBSD    TargetOS = "netbsd"
	OSDragonfly TargetOS = "dragonfly"
	OSSolaris   TargetOS = "solaris"
	OSPlan9     TargetOS = "plan9"
	OSIllumos   TargetOS = "illumos"
)

// TargetArch represents the architecture target for compilation or toolchain
type TargetArch string

const (
	ArchAmd64    TargetArch = "amd64"
	ArchArm64    TargetArch = "arm64"
	Arch386      TargetArch = "386"
	ArchArm      TargetArch = "arm"
	ArchMips     TargetArch = "mips"
	ArchMipsle   TargetArch = "mipsle"
	ArchMips64   TargetArch = "mips64"
	ArchMips64le TargetArch = "mips64le"
	ArchPpc64    TargetArch = "ppc64"
	ArchPpc64le  TargetArch = "ppc64le"
	ArchRiscv64  TargetArch = "riscv64"
	ArchS390x    TargetArch = "s390x"
	ArchWasm     TargetArch = "wasm"
)
