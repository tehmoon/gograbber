package main

import (
	"os/exec"
	"runtime"
	"bufio"
	"github.com/spf13/cobra"
	"os"
	"github.com/tehmoon/errors"
	"regexp"
	"net/url"
	"strings"
	"path"
	"path/filepath"
	"fmt"
)

type Scheduler struct {
	baseURL *url.URL
	directory string
	proxies []string
	limit chan struct{}
	done chan struct{}
	tracker chan struct{}
	sync chan error
	re *regexp.Regexp
}

type SchedulerConfig struct {
	Proxy string
	Directory string
	BaseURL string
	Threads int
}

func ParseURL(str string) (*url.URL, error) {
	u, err := url.Parse(str)
	if err != nil {
		return nil, errors.Wrap(err, "Error parsing URl")
	}

	if u.Scheme == "" {
		return nil, errors.New("Scheme is not defined")
	}

	if u.Host == "" {
		return nil, errors.New("Host is not defined")
	}

	u.RawQuery = ""
	u.ForceQuery = false
	u.Fragment = ""

	return u, nil
}

func NewScheduler(config *SchedulerConfig) (*Scheduler, error){
	t := config.Threads
	if t <= 0 {
		t = runtime.NumCPU()
	}

	var (
		err error
		proxies []string
	)

	if config.Proxy != "" {
		p, err := ParseURL(config.Proxy)
		if err != nil {
			return nil, errors.Wrapf(err, "Error parsing URL for %q", "proxy")
		}

		p.Scheme = "http"
		proxies = append(proxies, p.String())

		p.Scheme = "https"
		proxies = append(proxies, p.String())
	}

	u, err := ParseURL(config.BaseURL)
	if err != nil {
		return nil, errors.Wrapf(err, "Error parsing URL for %q", "base-url")
	}

	re := regexp.MustCompile(`^(/(?:\S+)?).*`)

	tracker := make(chan struct{}, t)
	done := make(chan struct{}, 0)
	sync := make(chan error, t)

	scheduler := &Scheduler{
		proxies: proxies,
		baseURL: u,
		limit: make(chan struct{}, t),
		tracker: tracker,
		sync: sync,
		done: done,
		re: re,
	}

	go func(tracker, done chan struct{}, sync chan error) {
		for {
			select {
				case <- tracker:
					<- sync
				case <- done:
					break
			}
		}
	}(tracker, done, sync)

	return scheduler, nil
}

func captureAsync(base, targetPath, directory string, proxies []string, done chan struct{}, sync chan error) {
	re := regexp.MustCompile(`/`)
	proxyServer := ""

	output := fmt.Sprintf("%s.pdf", filepath.Join(directory, re.ReplaceAllLiteralString(targetPath, "_")))
	print2pdf := fmt.Sprintf("--print-to-pdf=%s", output)

	if proxies != nil {
		if len(proxies) > 0 {
			proxyServer = fmt.Sprintf(`--proxy-server=%s`, strings.Join(proxies, ";"))
		}
	}

	command := []string{"chromium", "--headless", "--temp-profile", "--no-gpu", proxyServer, print2pdf, path.Join(base, targetPath),}
	cmd := exec.Command(command[0], command[1:]...)

	fmt.Fprintf(os.Stdout, "Printing to pdf for url %q\n", path.Join(base, targetPath))

	err := cmd.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%q\n", errors.Wrapf(err, "Error printing to pdf for url %q", path.Join(base, targetPath)).Error())
	}

	sync <- nil
	<- done
}

func (s Scheduler) Process(line string) {
	matches := s.re.FindStringSubmatch(line)
	if len(matches) != 2 {
		return
	}

	s.tracker <- struct{}{}
	s.limit <- struct{}{}

	go captureAsync(s.baseURL.String(), filepath.Join("/", matches[1]), s.directory, s.proxies, s.limit, s.sync)
}

// TODO: return error
func (s Scheduler) Wait() (error) {
	s.done <- struct{}{}

	return nil
}

func main() {
	root := &cobra.Command{
		Use: os.Args[0],
		Args: cobra.MaximumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) (error) {
			d, err := cmd.Flags().GetString("directory")
			if err != nil {
				return errors.Wrapf(err, "Error in flag %q", "-d")
			}

			u, err := cmd.Flags().GetString("base-url")
			if err != nil {
				return errors.Wrapf(err, "Error in flag %q", "-u")
			}

			p, err := cmd.Flags().GetString("proxy")
			if err != nil {
				return errors.Wrapf(err, "Error in flag %q", "-p")
			}

			t, err := cmd.Flags().GetInt("threads")
			if err != nil {
				return errors.Wrapf(err, "Error in flag %q", "-t")
			}

			if u == "" {
				return errors.Errorf("Flag %q is mandatory", "-u")
			}

			scheduler, err := NewScheduler(&SchedulerConfig{
				Proxy: p,
				BaseURL: u,
				Directory: d,
				Threads: t,
			})
			if err != nil {
				return errors.Wrap(err, "Error creating scheduler")
			}

			scheduler.Process("/")

			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				scheduler.Process(scanner.Text())
			}

			// TODO: return error
			scheduler.Wait()

			if err := scanner.Err(); err != nil {
				return errors.Wrap(err, "Error reading standard input")
			}

			return nil
		},
	}

	root.Flags().StringP("base-url", "u", "", "Base URL to prepand.")
	root.Flags().StringP("proxy", "p", "", "URL to http/https proxy.")
	root.Flags().StringP("directory", "d", ".", "Directory were to store the screenshots.")
	root.Flags().IntP("threads", "t", runtime.NumCPU(), "Number of concurrent running instances.")

	err := root.Execute()
	if err != nil {
		os.Exit(2)
	}
}
