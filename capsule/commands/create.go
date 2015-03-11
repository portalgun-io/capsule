package commands

import (
	"flag"
	"net/rpc"
	"os"
	"path/filepath"
	"time"

	"github.com/hamo/capsule/instance"
)

var (
	GlobalCapsuledDir  = filepath.Join("usr", "lib", "capsule", "libexec")
	GlobalCapsuledPath = filepath.Join(GlobalCapsuledDir, "capsuled")
)

var createCommand CapsuleCommand = CapsuleCommand{
	Handler:     cmdCreate,
	Description: "Create a new capsule",
}

func init() {
	RegisterCommand("create", &createCommand)
}

func cmdCreate(args []string, cmdEnv *CommandEnv) error {
	logger := cmdEnv.Logger

	createFlag := flag.NewFlagSet("create command", flag.ExitOnError)

	var (
		flInstanceName string
		flKernelName   string

		flMemorySize int

		flCmdline string
	)

	createFlag.StringVar(&flInstanceName, "name", "", "instance name")
	createFlag.StringVar(&flKernelName, "kernel", "", "kernel name")

	createFlag.IntVar(&flMemorySize, "m", 512, "memory size")

	createFlag.StringVar(&flCmdline, "cmdline", "", "cmdline")

	createFlag.Parse(args)

	if flInstanceName == "" {
		logger.Fatalln("name")
	}
	if flKernelName == "" {
		logger.Fatalln("kernel")
	}

	instancesCatalog, err := cmdEnv.BaseCatalog.Dir("instances")
	if err != nil {
		logger.Fatal(err)
	}

	if _, err := instancesCatalog.TryDir(flInstanceName); err == nil {
		logger.Fatalf("%s exists\n", flInstanceName)
	}

	myInstanceCatalog, err := instancesCatalog.Dir(flInstanceName)
	if err != nil {
		logger.Fatalln("create instance catalog failed.")
	}

	kernelsCatalog, err := cmdEnv.BaseCatalog.Dir("kernels")
	if err != nil {
		logger.Fatalln("can not read kernel catalog")
	}

	myKernelCatalog, err := kernelsCatalog.TryDir(flKernelName)
	if err != nil {
		logger.Fatalf("can not find kernel named %s.", flKernelName)
	}

	i := instance.New(flInstanceName)
	i.Kernel = flKernelName
	i.Cmdline = flCmdline

	if flMemorySize < 512 {
		// FIXME: support huge initrd
		cmdEnv.Logger.Infoln("workaround: force memory size to 512M. ")
		i.MemorySize = 512
	} else {
		i.MemorySize = flMemorySize
	}

	i.InstanceCatalog = myInstanceCatalog
	i.KernelCatalog = myKernelCatalog

	// Find capsuled
	// it should be either placed $CAPSULE_ROOT/capsuled/capsuled or
	// /usr/lib/capsule/libexec/capusled
	// Pass its parent dir to qemu
	capsuledDir := filepath.Join(cmdEnv.BaseCatalog.Path, "capsuled")
	if fi, err := os.Stat(filepath.Join(capsuledDir, "capsuled")); err != nil || !fi.Mode().IsRegular() {
		// Can not find capsuled at $CAPSULE_ROOT/capsuled/capsuled
		if fi, err := os.Stat(GlobalCapsuledPath); err != nil || !fi.Mode().IsRegular() {
			cmdEnv.Logger.Fatalln("can not find capsuled")
		} else {
			capsuledDir = GlobalCapsuledDir
		}
	}
	i.SysinitDir = capsuledDir

	err = i.Create()
	if err != nil {
		panic(err)
	}

	// FIXME: ugly workaround to wait for qemu start
	time.Sleep(2 * time.Second)

	controlSock, err := myInstanceCatalog.File("control.sock", false)
	if err != nil {
		panic(err)
	}

	client, err := rpc.Dial("unix", controlSock)
	if err != nil {
		panic(err)
	}

	var d time.Duration
	err = client.Call("Server.Alive", struct{}{}, &d)
	if err != nil {
		panic(err)
	}

	cmdEnv.Logger.Infoln("Yoo, we are alive.")

	return nil
}

func createUsage() {

}
