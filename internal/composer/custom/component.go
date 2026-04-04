package custom

import (
	"net/http"
	"slices"

	"github.com/trebent/kerberos/internal/composer"
	adminapi "github.com/trebent/kerberos/internal/oapi/admin"
)

type (
	custom struct {
		http.Handler

		all   []composer.FlowComponent
		first composer.FlowComponent
	}
	Ordered interface {
		composer.FlowComponent
		Order() int
	}
)

var _ composer.FlowComponent = (*custom)(nil)

func NewComponent(components ...composer.FlowComponent) composer.FlowComponent {
	if len(components) == 0 {
		return &custom{}
	}

	ordered := make([]Ordered, 0)
	unordered := make([]composer.FlowComponent, 0)
	for _, component := range components {
		ord, ok := component.(Ordered)
		if ok {
			ordered = append(ordered, ord)
		} else {
			unordered = append(unordered, component)
		}
	}

	slices.SortFunc(ordered, func(one Ordered, two Ordered) int {
		return one.Order() - two.Order()
	})

	all := make([]composer.FlowComponent, 0, len(ordered)+len(unordered))
	for _, ord := range ordered {
		all = append(all, ord)
	}
	all = append(all, unordered...)

	for i, component := range all {
		// As long as not last element.
		if i == len(all)-1 {
			break
		}

		// The current component's next caller is the next sorted component.
		component.Next(all[i+1])
	}

	return &custom{first: all[0], all: all}
}

func (c *custom) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// By calling first, the full custom chain will be executed as the linked list is set up
	// in the component constructor function, or the next component if the custom setup was
	// empty.
	c.first.ServeHTTP(w, req)
}

func (c *custom) Next(next composer.FlowComponent) {
	// first will be nil if 0 components were given to the custom constructor, use next
	// as first in this case.
	if c.first == nil {
		c.first = next
	} else {
		// If there exists a first component, assign the last component in the ordered
		// component slice the input next component.
		c.all[len(c.all)-1].Next(next)
	}
}

// GetMeta implements [composer.FlowComponent].
func (c *custom) GetMeta() []adminapi.FlowMeta {
	// Skip adding custom component metadata for now, since it's not actually manipulating requests
	// in any way, other than forwarding the call to the first component according to its order.
	return c.first.GetMeta()
}
