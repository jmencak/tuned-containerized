package main

import (
	"bufio"   // scanner
	"flag"    // command-line options parsing
	"fmt"     // Printf()
	"os"      // os.Exit(), os.Signal, os.Stderr, ...
	"strings" // strings.Join()
	"time"    // time.Sleep()

	"github.com/fsnotify/fsnotify"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

/* Types */
type arrayFlags []string

/* Constants */
const (
	probe_seconds = 5
	PNAME         = "tuned-wait"
)

/* Global variables */
var (
	boolDumpNodeLabels = flag.Bool("dump-node-labels", false, "dump node labels and quit")
	fileNodeLabels     = "/tmp/ocp-node-labels.cfg"
	fileWatch          arrayFlags
)

/* Functions */
func (a *arrayFlags) String() string {
	return strings.Join(*a, ",")
}

func (a *arrayFlags) Set(value string) error {
	*a = append(*a, value)
	return nil
}

func parseCmdOpts() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <NODE>\n", PNAME)
		fmt.Fprintf(os.Stderr, "Example: %s b1.lan\n\n", PNAME)
		fmt.Fprintf(os.Stderr, "Options:\n")

		flag.PrintDefaults()
	}

	flag.Var(&fileWatch, "watch", "Files/directories to watch for changes.")
	flag.StringVar(&fileNodeLabels, "l", fileNodeLabels, "File to dump node-labels to for tuned.")
	flag.Parse() // to execute the command-line parsing
}

func nodeLabelsGet(clientset *kubernetes.Clientset, nodeName string) (nodeLabels map[string]string) {
	node, err := clientset.CoreV1().Nodes().Get(nodeName, metav1.GetOptions{})
	if err != nil {
		panic(err.Error())
	}
	if errors.IsNotFound(err) {
		fmt.Printf("node %s not found\n", nodeName)
	} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
		fmt.Printf("error getting node %v\n", statusError.ErrStatus.Message)
	} else if err != nil {
		panic(err.Error())
	}

	return node.Labels
}

func nodeLabelsRead() map[string]string {
	nodeLabels := make(map[string]string)

	if _, err := os.Stat(fileNodeLabels); os.IsNotExist(err) {
		/* node labels file does not exist */
		return nil
	}

	f, err := os.Open(fileNodeLabels)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening node labels file %s: %v\n", fileNodeLabels, err)
		os.Exit(1)
	}
	defer f.Close()

	var scanner = bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		// fmt.Fprintf(os.Stderr, "|%s|\n", line)
		if equal := strings.Index(line, "="); equal >= 0 {
			if key := strings.TrimSpace(line[:equal]); len(key) > 0 {
				value := line[equal+1:]
				nodeLabels[key] = value
				// fmt.Fprintf(os.Stderr, "|%s|%s|\n", key, value)
			}
		} else {
			/* no '=' sign found */
			fmt.Fprintf(os.Stderr, "Invalid key=value pair in node labels file %s: %s\n", fileNodeLabels, line)
			os.Exit(1)
		}
	}

	// fmt.Fprintf(os.Stderr, "|%v|\n", nodeLabels)
	return nodeLabels
}

func nodeLabelsDump(nodeLabels map[string]string) {
	f, err := os.Create(fileNodeLabels)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating node labels file %s: %v\n", fileNodeLabels, err)
		os.Exit(1)
	}
	defer f.Close()

	for key, value := range nodeLabels {
		_, err := f.WriteString(fmt.Sprintf("%s=%s\n", key, value))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing to node labels file %s: %v\n", fileNodeLabels, err)
			os.Exit(1)
		}
	}
	f.Sync()
}

func nodeLabelsCompare(nodeLabelsOld map[string]string, nodeLabelsNew map[string]string) bool {
	/* Found the node */
	if nodeLabelsOld == nil {
		/* no node labels defined yet */
		return false
	}
	if len(nodeLabelsOld) != len(nodeLabelsNew) {
		fmt.Printf("node labels changed, quitting...\n")
		nodeLabelsDump(nodeLabelsNew)
		os.Exit(0)
	}
	for key, value := range nodeLabelsNew {
		if nodeLabelsOld[key] != value {
			fmt.Printf("node label[%s] == %s (old value: %s), quitting...\n", key, value, nodeLabelsOld[key])
			nodeLabelsDump(nodeLabelsNew)
			os.Exit(0)
		}
	}
	return true
}

func watcherAdd(watcher *fsnotify.Watcher, file string) {
	err := watcher.Add(file)
	if err != nil {
		panic(err.Error)
	}
}

func main() {
	parseCmdOpts()

	if len(flag.Args()) != 1 {
		flag.Usage()
		os.Exit(1)
	}

	nodeName := flag.Args()[0]

	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	if *boolDumpNodeLabels {
		nodeLabelsDump(nodeLabelsGet(clientset, nodeName))
		os.Exit(0)
	}

	nodeLabelsOld := nodeLabelsRead()
	ticker := time.NewTicker(time.Second * probe_seconds)
	go func() {
		for range ticker.C {
			nodeLabels := nodeLabelsGet(clientset, nodeName)
			nodeLabelsCompare(nodeLabelsOld, nodeLabels)
		}
	}()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		panic(err.Error())
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				fmt.Println("event:", event)
				if event.Op&fsnotify.Write == fsnotify.Write ||
					event.Op&fsnotify.Remove == fsnotify.Remove ||
					event.Op&fsnotify.Create == fsnotify.Create {
					fmt.Printf("%s: modified file: %s\n", PNAME, event.Name)
					nodeLabelsDump(nodeLabelsGet(clientset, nodeName))
					os.Exit(0)
				}
			case err := <-watcher.Errors:
				fmt.Println("error:", err)
			}
		}
	}()

	for _, element := range fileWatch {
		watcherAdd(watcher, element)
	}
	<-done
}
