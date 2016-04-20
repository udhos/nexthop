package command

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type sortByCommitId []string

func (s sortByCommitId) Len() int {
	return len(s)
}
func (s sortByCommitId) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s sortByCommitId) Less(i, j int) bool {
	s1 := s[i]
	id1, err1 := ExtractCommitIdFromFilename(s1)
	if err1 != nil {
		log.Printf("sortByCommitId.Less: error parsing config file path: '%s': %v", s1, err1)
	}
	s2 := s[j]
	id2, err2 := ExtractCommitIdFromFilename(s2)
	if err2 != nil {
		log.Printf("sortByCommitId.Less: error parsing config file path: '%s': %v", s2, err2)
	}
	return id1 < id2
}

func getConfigPath(configPathPrefix, id string) string {
	return fmt.Sprintf("%s%s", configPathPrefix, id)
}

func ExtractCommitIdFromFilename(filename string) (int, error) {
	lastDot := strings.LastIndexByte(filename, '.')
	commitId := filename[lastDot+1:]
	id, err := strconv.Atoi(commitId)
	if err != nil {
		return -1, fmt.Errorf("extractCommitIdFromFilename: error parsing filename [%s]: %v", filename, err)
	}

	return id, nil
}

func FindLastConfig(configPathPrefix string) (string, error) {
	log.Printf("FindLastConfig: configuration path prefix: %s", configPathPrefix)

	dirname, matches, err := ListConfig(configPathPrefix, false)
	if err != nil {
		return "", err
	}

	m := len(matches)

	log.Printf("FindLastConfig: found %d matching files: %v", m, matches)

	if m < 1 {
		return "", fmt.Errorf("FindLastConfig: no config file found for prefix: %s", configPathPrefix)
	}

	lastConfig := filepath.Join(dirname, matches[m-1])

	return lastConfig, nil
}

func ListConfig(configPathPrefix string, reverse bool) (string, []string, error) {
	//log.Printf("FindLastConfig: configuration path prefix: %s", configPathPrefix)

	dirname := filepath.Dir(configPathPrefix)

	dir, err := os.Open(dirname)
	if err != nil {
		return "", nil, fmt.Errorf("FindLastConfig: error opening dir '%s': %v", dirname, err)
	}

	names, e := dir.Readdirnames(0)
	if e != nil {
		return "", nil, fmt.Errorf("FindLastConfig: error reading dir '%s': %v", dirname, e)
	}

	dir.Close()

	//log.Printf("FindLastConfig: found %d files: %v", len(names), names)

	basename := filepath.Base(configPathPrefix)

	// filter prefix
	matches := names[:0]
	for _, x := range names {
		//log.Printf("FindLastConfig: x=[%s] prefix=[%s]", x, basename)
		if strings.HasPrefix(x, basename) {
			matches = append(matches, x)
		}
	}

	if reverse {
		sort.Sort(sort.Reverse(sortByCommitId(matches)))
	} else {
		sort.Sort(sortByCommitId(matches))
	}

	return dirname, matches, nil
}

type configLineWriter struct {
	writer *bufio.Writer
}

func (w *configLineWriter) WriteLine(s string) (int, error) {
	return w.writer.WriteString(fmt.Sprintf("%s\n", s))
}

func bogusConfigPathPrefix(prefix string) bool {
	return strings.HasPrefix(prefix, "BOGUS")
}

func SaveNewConfig(configPathPrefix string, root *ConfNode, maxFiles int) (string, error) {

	if bogusConfigPathPrefix(configPathPrefix) {
		return "", fmt.Errorf("SaveNewConfig: refusing to save to bogus config prefix: [%s]", configPathPrefix)
	}

	lastConfig, err1 := FindLastConfig(configPathPrefix)
	if err1 != nil {
		log.Printf("SaveNewConfig: error reading config: [%s]: %v", configPathPrefix, err1)
	}

	id, err2 := ExtractCommitIdFromFilename(lastConfig)
	if err2 != nil {
		log.Printf("SaveNewConfig: error parsing config path: [%s]: %v", lastConfig, err2)
	}

	newCommitId := id + 1

	newFilepath := fmt.Sprintf("%s%d", configPathPrefix, newCommitId)

	log.Printf("SaveNewConfig: newPath=[%s]", newFilepath)

	if _, err := os.Stat(newFilepath); err == nil {
		return "", fmt.Errorf("SaveNewConfig: new file exists: [%s]", newFilepath)
	}

	f, err3 := os.Create(newFilepath)
	if err3 != nil {
		return "", fmt.Errorf("SaveNewConfig: error creating file: [%s]: %v", newFilepath, err3)
	}

	w := bufio.NewWriter(f)
	cw := configLineWriter{w}

	if err := WriteConfig(root, &cw, false); err != nil {
		return "", fmt.Errorf("SaveNewConfig: error writing file: [%s]: %v", newFilepath, err)
	}

	if err := w.Flush(); err != nil {
		return "", fmt.Errorf("SaveNewConfig: error flushing file: [%s]: %v", newFilepath, err)
	}

	if err := f.Close(); err != nil {
		return "", fmt.Errorf("SaveNewConfig: error closing file: [%s]: %v", newFilepath, err)
	}

	eraseOldFiles(configPathPrefix, root, maxFiles)

	return newFilepath, nil
}

func eraseOldFiles(configPathPrefix string, root *ConfNode, maxFiles int) {

	if maxFiles < 1 {
		return
	}

	dirname, matches, err := ListConfig(configPathPrefix, false)
	if err != nil {
		log.Printf("eraseOldFiles: %v", err)
		return
	}

	totalFiles := len(matches)

	toDelete := totalFiles - maxFiles
	if toDelete < 1 {
		log.Printf("eraseOldFiles: nothing to delete existing=%d <= max=%d", totalFiles, maxFiles)
		return
	}

	for i := 0; i < toDelete; i++ {
		path := filepath.Join(dirname, matches[i])
		log.Printf("eraseOldFiles: delete: [%s]", path)
		if err := os.Remove(path); err != nil {
			log.Printf("eraseOldFiles: delete: error: [%s]: %v", path, err)
		}
	}
}

type LineWriter interface {
	WriteLine(s string) (int, error)
}

func WriteConfig(node *ConfNode, w LineWriter, infoType bool) error {

	// show node children
	if len(node.Children) == 0 {
		var line string
		if infoType {
			line = fmt.Sprintf("C %s", node.Path)
		} else {
			line = node.Path
		}
		size := len(line)
		count, err := w.WriteLine(line)
		if count < size || err != nil {
			return fmt.Errorf("writeConfig: error: write=%d < size=%d: %v", count, size, err)
		}
		return nil
	}

	// scan children
	for _, n := range node.Children {
		if err := WriteConfig(n, w, infoType); err != nil {
			return err
		}
	}

	return nil
}

func LoadConfig(ctx ConfContext, path string, c CmdClient, abortOnError bool) (int, error) {

	goodLines := 0

	consume := func(line string) error {
		if err := Dispatch(ctx, line, c, CONF, false); err != nil {
			return fmt.Errorf("LoadConfig: dispatch error: %v", err)
		}
		goodLines++
		return nil // no error
	}

	err := scanConfigFile(consume, path, abortOnError)

	return goodLines, err
}

type lineConsumerFunc func(line string) error

func scanConfigFile(consumer lineConsumerFunc, path string, abortOnError bool) error {
	f, err1 := os.Open(path)
	if err1 != nil {
		return fmt.Errorf("scanConfigFile: error opening config file: [%s]: %v", path, err1)
	}

	defer f.Close()

	scanner := bufio.NewScanner(f)

	var lastErr error
	i := 0

	for scanner.Scan() {
		i++
		line := scanner.Text()
		if err := consumer(line); err != nil {
			lastErr = fmt.Errorf("scanConfigFile: error consuming line %d [%s] from file: [%s]: %v", i, line, path, err)
			log.Printf("%v", lastErr)
			if abortOnError {
				return lastErr
			}
		}
	}

	if err := scanner.Err(); err != nil {
		lastErr = fmt.Errorf("scanConfigFile: error scanning config file: [%s]: %v", path, err)
	}

	return lastErr
}
