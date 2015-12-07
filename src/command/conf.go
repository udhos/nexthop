package command

import (
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
	lastDot1 := strings.LastIndexByte(s1, '.')
	commitId1 := s1[lastDot1+1:]
	id1, err1 := strconv.Atoi(commitId1)
	if err1 != nil {
		log.Printf("sortByCommitId.Less: error parsing config file path: '%s': %v", s1, err1)
	}
	s2 := s[j]
	lastDot2 := strings.LastIndexByte(s2, '.')
	commitId2 := s2[lastDot2+1:]
	id2, err2 := strconv.Atoi(commitId2)
	if err2 != nil {
		log.Printf("sortByCommitId.Less: error parsing config file path: '%s': %v", s2, err2)
	}
	return id1 < id2
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
