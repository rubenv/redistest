// Spawns a Redis server. Ideal for unit tests where you want a clean instance
// each time. Then clean up afterwards.
//
// Requires Redis to be installed on your system (but it doesn't have to be running).
package redistest

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"syscall"
	"time"

	"github.com/gomodule/redigo/redis"
)

type Redis struct {
	dir string
	cmd *exec.Cmd

	// A redis pool pre-configured to talk to the redis server
	Pool *redis.Pool

	// Network protocol to pass to redis.Dial()
	Network string

	// Address to pass to redis.Dial()
	Address string

	stderr io.ReadCloser
	stdout io.ReadCloser
}

// Start a new Redis database, on temporary storage.
//
// This database has persistance disabled for performance, so it might run faster
// than your production database. This makes it less reliable in case of system
// crashes, but we don't care about that anyway during unit testing.
//
// Use the Pool field to access the database connection
func Start() (*Redis, error) {
	// Prepare data directory
	dir, err := ioutil.TempDir("", "redistest")
	if err != nil {
		return nil, err
	}

	sockDir := path.Join(dir, "sock")
	err = os.MkdirAll(sockDir, 0711)
	if err != nil {
		return nil, err
	}

	// Config file
	//
	// We're always using unix sockets, but if someone wants to make this
	// conditional and use TCP sockets on Windows: feel free to send a PR.
	network := "unix"
	address := fmt.Sprintf("%s/redis.sock", sockDir)
	configFile := path.Join(dir, "redis.cnf")
	err = ioutil.WriteFile(configFile, []byte(fmt.Sprintf(`
port 0
unixsocket %s
appendonly no
save ""
`, address)), 0644)
	if err != nil {
		return nil, err
	}

	// Find executables root path
	binPath, err := findBinPath()
	if err != nil {
		return nil, err
	}

	// Start Redis
	cmd := exec.Command(path.Join(binPath, "redis-server"),
		configFile,
	)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stderr.Close()
		return nil, err
	}

	err = cmd.Start()
	if err != nil {
		return nil, abort("Failed to start Redis", cmd, stderr, stdout, err)
	}

	// Connect to Redis
	pool := &redis.Pool{
		Dial: func() (redis.Conn, error) {
			return redis.Dial(network, address)
		},
	}

	err = retry(func() error {
		conn := pool.Get()
		defer conn.Close()

		_, err := conn.Do("PING")
		return err
	}, 1000, 10*time.Millisecond)
	if err != nil {
		return nil, abort("Failed to connect to DB", cmd, stderr, stdout, err)
	}

	pg := &Redis{
		cmd: cmd,
		dir: dir,

		Pool:    pool,
		Network: network,
		Address: address,

		stderr: stderr,
		stdout: stdout,
	}

	return pg, nil
}

// Stop the database and remove storage files.
func (s *Redis) Stop() error {
	if s == nil {
		return nil
	}

	defer func() {
		// Always try to remove it
		os.RemoveAll(s.dir)
	}()

	err := s.cmd.Process.Signal(os.Interrupt)
	if err != nil {
		return err
	}

	err = s.cmd.Wait()
	if err != nil {
		return err
	}

	if s.stderr != nil {
		s.stderr.Close()
	}

	if s.stdout != nil {
		s.stdout.Close()
	}

	return nil
}

// Hang the server, good for testing blocked connections
func (s *Redis) Freeze() {
	if s.cmd != nil {
		s.cmd.Process.Signal(syscall.SIGSTOP)
	}
}

// Resume the server
func (s *Redis) Continue() {
	if s.cmd != nil {
		s.cmd.Process.Signal(syscall.SIGCONT)
	}
}

// Needed because Ubuntu doesn't put initdb in $PATH
func findBinPath() (string, error) {
	// In $PATH (e.g. Fedora) great!
	p, err := exec.LookPath("redis-server")
	if err == nil {
		return path.Dir(p), nil
	}

	return "", fmt.Errorf("Did not find Redis executables installed")
}

func retry(fn func() error, attempts int, interval time.Duration) error {
	for {
		err := fn()
		if err == nil {
			return nil
		}

		attempts -= 1
		if attempts <= 0 {
			return err
		}

		time.Sleep(interval)
	}
}

func abort(msg string, cmd *exec.Cmd, stderr, stdout io.ReadCloser, err error) error {
	cmd.Process.Signal(os.Interrupt)
	cmd.Wait()

	serr, _ := ioutil.ReadAll(stderr)
	sout, _ := ioutil.ReadAll(stdout)
	stderr.Close()
	stdout.Close()
	return fmt.Errorf("%s: %s\nOUT: %s\nERR: %s", msg, err, string(sout), string(serr))
}
