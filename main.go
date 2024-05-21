package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"path/filepath"
	"io/ioutil"
)

func main() {
	switch os.Args[1] {
	case "run":
		run()
	case "child":
		child()
		
	default:
		panic("bad command")
	}
}

func run() {
	fmt.Printf("Running %v as %d\n", os.Args[2:], os.Getpid())

	cmd := exec.Command("/proc/self/exe", append([]string{"child"}, os.Args[2:]...)...) // /proc/self/exe is a symbolic link to the current process
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS, // UTS namespace, PID namespace, Mount namespace
		Unshareflags: syscall.CLONE_NEWNS, // container 바깥에 mount namespace를 공유하지 않음. 바깥에서 "mount | grep proc" 명령어를 실행하면 컨테이너 내부에서 mount된 proc이 보이지 않음
	}
	cmd.Run()
}

func child() {
	fmt.Printf("Running %v as %d\n", os.Args[2:], os.Getpid())

	cg()

	cmd := exec.Command(os.Args[2], os.Args[3:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	must(syscall.Sethostname([]byte("container"))) // 새로운 namespace에 들어왔으므로 hostname을 변경해도 host의 hostname은 변경되지 않음
	must(syscall.Chroot("/home/ubuntu/container")) // chroot를 통해 root directory를 변경
	must(syscall.Chdir("/"))
	must(syscall.Mount("proc", "proc", "proc", 0, "")) // src(device), target(dir), fstype, flags, data (src를 target에 mount)

	must(cmd.Run())

	must(syscall.Unmount("/proc", 0)) // unmount proc

}

func cg() {
	cgroups := "/sys/fs/cgroup/"
	pids := filepath.Join(cgroups, "pids")
	err := os.Mkdir(filepath.Join(pids, "container"), 0755) // 0755: rwxr-xr-x
	if err != nil && !os.IsExist(err) {
		panic(err)
	}
	must(ioutil.WriteFile(filepath.Join(pids, "container/pids.max"), []byte("20"), 0700)) // 0700: rwx------
	// Removes the new cgroup in plcae after the container exits
	must(ioutil.WriteFile(filepath.Join(pids, "container/notify_on_release"), []byte("1"), 0700))
	must(ioutil.WriteFile(filepath.Join(pids, "container/cgroup.procs"), []byte(fmt.Sprintf("%d", os.Getpid())), 0700))
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}