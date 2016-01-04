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
	id1, err1 := extractCommitIdFromFilename(s1)
	if err1 != nil {
		log.Printf("sortByCommitId.Less: error parsing config file path: '%s': %v", s1, err1)
	}
	s2 := s[j]
	id2, err2 := extractCommitIdFromFilename(s2)
	if err2 != nil {
		log.Printf("sortByCommitId.Less: error parsing config file path: '%s': %v", s2, err2)
	}
	return id1 < id2
}

func extractCommitIdFromFilename(filename string) (int, error) {
	lastDot := strings.LastIndexByte(filename, '.')
	commitId := filename[lastDot+1:]
	id, err := strconv.Atoi(commitId)
	if err != nil {
		return 0, fmt.Errorf("extractCommitIdFromFilename: error parsing filename [%s]: %v", filename, err)
	}

	return id, nil
}

func FindLastConfig(configPathPrefix string) (string, error) {
	log.Printf("FindLastConfig: configuration path prefix: %s", configPathPrefix)

	dirname := filepath.Dir(configPathPrefix)

	dir, err := os.Open(dirname)
	if err != nil {
		return "", fmt.Errorf("FindLastConfig: error opening dir '%s': %v", dirname, err)
	}

	names, e := dir.Readdirnames(0)
	if e != nil {
		return "", fmt.Errorf("FindLastConfig: error reading dir '%s': %v", dirname, e)
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

	sort.Sort(sortByCommitId(matches))

	m := len(matches)

	log.Printf("FindLastConfig: found %d matching files: %v", m, matches)

	if m < 1 {
		return "", fmt.Errorf("FindLastConfig: no config file found for prefix: %s", configPathPrefix)
	}

	lastConfig := names[m-1]

	return lastConfig, nil
}

func SaveNewConfig(configPathPrefix string, root *ConfNode) error {
	log.Printf("SaveNewConfig: prefix=[%s]", configPathPrefix)

	lastConfig, err1 := FindLastConfig(configPathPrefix)
	if err1 != nil {
		return fmt.Errorf("SaveNewConfig: error reading config: [%s]: %v", configPathPrefix, err1)
	}

	id, err2 := extractCommitIdFromFilename(lastConfig)
	if err2 != nil {
		return fmt.Errorf("SaveNewConfig: error parsing config path: [%s]: %v", lastConfig, err2)
	}

	newCommitId := id + 1

	newFilepath := fmt.Sprintf("%s%d", configPathPrefix, newCommitId)

	log.Printf("SaveNewConfig: newPath=[%s]", newFilepath)

	f, err3 := os.Create(newFilepath)
	if err3 != nil {
		return fmt.Errorf("SaveNewConfig: error creating file: [%s]: %v", newFilepath, err3)
	}

	w := bufio.NewWriter(f)

	if err := writeConfig(root, w); err != nil {
		return fmt.Errorf("SaveNewConfig: error writing file: [%s]: %v", newFilepath, err)
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf("SaveNewConfig: error flushing file: [%s]: %v", newFilepath, err)
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("SaveNewConfig: error closing file: [%s]: %v", newFilepath, err)
	}

	return nil
}

type StringWriter interface {
	WriteString(s string) (int, error)
}

func writeConfig(node *ConfNode, w StringWriter) error {

	if len(node.Value) == 0 && len(node.Children) == 0 {
		line := fmt.Sprintf("%s\n", node.Path)
		size := len(line)
		count, err := w.WriteString(line)
		if count < size || err != nil {
			return fmt.Errorf("writeConfig: error: write=%d < size=%d: %v", count, size, err)
		}
		return nil
	}

	// show node values
	for _, v := range node.Value {
		line := fmt.Sprintf("%s %s\n", node.Path, v)
		size := len(line)
		count, err := w.WriteString(line)
		if count < size || err != nil {
			return fmt.Errorf("writeConfig: error: write=%d < size=%d: %v", count, size, err)
		}
	}

	// scan children
	for _, n := range node.Children {
		if err := writeConfig(n, w); err != nil {
			return err
		}
	}

	return nil
}
