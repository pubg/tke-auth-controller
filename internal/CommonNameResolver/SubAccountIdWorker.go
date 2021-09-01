package CommonNameResolver

import (
	"example.com/tke-auth-controller/internal"
	"github.com/pkg/errors"
	"sync"
)

type CommonNameResolver struct {
	resolveWorkers map[string]CommonNameResolveWorker
}

type CommonNameResolveWorker interface {
	ValueType() string
	ResolveCommonNames(namesPtr []*internal.User) error
}

func NewCommonNameResolver() *CommonNameResolver {
	resolver := &CommonNameResolver{
		resolveWorkers: make(map[string]CommonNameResolveWorker),
	}

	return resolver
}

func (resolver *CommonNameResolver) AddWorker(worker CommonNameResolveWorker) {
	if worker == nil {
		panic("tried to add CommonNameResolveWorker but value is nil.")
	}

	valueType := worker.ValueType()
	resolver.resolveWorkers[valueType] = worker
}

func (resolver *CommonNameResolver) ResolveCommonNames(users []internal.User) error {
	UsersSortedByType := sortUsersByType(users)
	errs := make([]error, 0)
	waitGroup := sync.WaitGroup{}

	for valueType, users := range UsersSortedByType {
		worker, ok := resolver.resolveWorkers[valueType]
		if ok {
			waitGroup.Add(1)
			go func() {
				defer waitGroup.Done()
				err := worker.ResolveCommonNames(users)
				if err != nil {
					errs = append(errs, err)
				}
			}()
		}
	}

	waitGroup.Wait()

	if len(errs) > 0 {
		return errors.Errorf("multiple errors raised while resolving common Names... err: %s\n", errs)
	} else {
		return nil
	}
}

func sortUsersByType(users []internal.User) map[string][]*internal.User {
	ret := make(map[string][]*internal.User)

	for _, user := range users {
		if _, ok := ret[user.ValueType]; !ok {
			ret[user.ValueType] = make([]*internal.User, 1)
		}

		ret[user.ValueType] = append(ret[user.ValueType], &user)
	}

	return ret
}
