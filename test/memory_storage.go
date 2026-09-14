package test

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/samedi/caldav-go/data"
	"github.com/samedi/caldav-go/errs"
)

// NewMemoryStorage creates an in-memory storage to be used in unit tests. Contrary to `FakeStorage`,
// it does not touch the file system nor the globals, so several independent storages can be used
// at the same time (which is needed, for example, to test concurrent requests).
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{resources: make(map[string]string), nonCalendars: make(map[string]bool)}
}

type MemoryStorage struct {
	// resource path -> resource content. Paths ending with a "/" are collections.
	resources    map[string]string
	nonCalendars map[string]bool
}

// AddResource adds a resource with the given content to the storage. Paths ending
// with a "/" are added as collections.
func (s *MemoryStorage) AddResource(path, content string) {
	s.resources[path] = content
}

// AddNonCalendarCollection adds a collection that is not a calendar, like a calendar home set.
func (s *MemoryStorage) AddNonCalendarCollection(path string) {
	s.resources[path] = ""
	s.nonCalendars[path] = true
}

func (s *MemoryStorage) GetResources(rpath string, withChildren bool) ([]data.Resource, error) {
	content, found := s.resources[rpath]
	if !found {
		return nil, errs.ResourceNotFoundError
	}

	result := []data.Resource{s.newResource(rpath, content)}

	if withChildren && isCollectionPath(rpath) {
		for _, childPath := range s.childPaths(rpath) {
			result = append(result, s.newResource(childPath, s.resources[childPath]))
		}
	}

	return result, nil
}

func (s *MemoryStorage) GetResourcesByList(rpaths []string) ([]data.Resource, error) {
	result := []data.Resource{}

	for _, rpath := range rpaths {
		content, found := s.resources[rpath]
		if found {
			result = append(result, s.newResource(rpath, content))
		}
	}

	return result, nil
}

func (s *MemoryStorage) GetResourcesByFilters(rpath string, filters *data.ResourceFilter) ([]data.Resource, error) {
	result := []data.Resource{}

	for _, childPath := range s.childPaths(rpath) {
		resource := s.newResource(childPath, s.resources[childPath])

		if filters == nil || filters.Match(&resource) {
			result = append(result, resource)
		}
	}

	return result, nil
}

func (s *MemoryStorage) GetResource(rpath string) (*data.Resource, bool, error) {
	return s.GetShallowResource(rpath)
}

func (s *MemoryStorage) GetShallowResource(rpath string) (*data.Resource, bool, error) {
	content, found := s.resources[rpath]
	if !found {
		return nil, false, errs.ResourceNotFoundError
	}

	resource := s.newResource(rpath, content)
	return &resource, true, nil
}

func (s *MemoryStorage) CreateResource(rpath, content string) (*data.Resource, error) {
	if _, found := s.resources[rpath]; found {
		return nil, errs.ResourceAlreadyExistsError
	}

	s.resources[rpath] = content
	resource := s.newResource(rpath, content)

	return &resource, nil
}

func (s *MemoryStorage) UpdateResource(rpath, content string) (*data.Resource, error) {
	if _, found := s.resources[rpath]; !found {
		return nil, errs.ResourceNotFoundError
	}

	s.resources[rpath] = content
	resource := s.newResource(rpath, content)

	return &resource, nil
}

func (s *MemoryStorage) DeleteResource(rpath string) error {
	if _, found := s.resources[rpath]; !found {
		return errs.ResourceNotFoundError
	}

	delete(s.resources, rpath)

	return nil
}

func (s *MemoryStorage) newResource(rpath, content string) data.Resource {
	return data.NewResource(rpath, memoryResourceAdapter{content, isCollectionPath(rpath), !s.nonCalendars[rpath]})
}

// Returns the direct children paths of the collection in `rpath`, sorted to keep the results stable.
func (s *MemoryStorage) childPaths(rpath string) []string {
	if !isCollectionPath(rpath) {
		return nil
	}

	paths := []string{}
	for path := range s.resources {
		if path == rpath || !strings.HasPrefix(path, rpath) {
			continue
		}

		if strings.Contains(strings.TrimSuffix(strings.TrimPrefix(path, rpath), "/"), "/") {
			continue
		}

		paths = append(paths, path)
	}
	sort.Strings(paths)

	return paths
}

func isCollectionPath(path string) bool {
	return strings.HasSuffix(path, "/")
}

type memoryResourceAdapter struct {
	content    string
	collection bool
	calendar   bool
}

func (a memoryResourceAdapter) IsCollection() bool {
	return a.collection
}

func (a memoryResourceAdapter) IsCalendar() bool {
	return a.calendar
}

func (a memoryResourceAdapter) CalculateEtag() string {
	return fmt.Sprintf(`"%x"`, len(a.content))
}

func (a memoryResourceAdapter) GetContent() string {
	return a.content
}

func (a memoryResourceAdapter) GetContentSize() int64 {
	return int64(len(a.content))
}

func (a memoryResourceAdapter) GetModTime() time.Time {
	return time.Unix(0, 0).UTC()
}
