package myrasec

import (
	"fmt"
	"runtime"
	"strconv"

	mgo "github.com/Myra-Security-GmbH/myrasec-go/v2"
	"github.com/chenrui333/terraformer/terraformutils"
	"golang.org/x/sync/errgroup"
)

// DomainGenerator
type DomainGenerator struct {
	MyrasecService
}

// createDomainResource
func (g *DomainGenerator) createDomainResource(_ *mgo.API, domain mgo.Domain) error {
	d := terraformutils.NewResource(
		strconv.Itoa(domain.ID),
		fmt.Sprintf("%s_%d", domain.Name, domain.ID),
		"myrasec_domain",
		"myrasec",
		map[string]string{},
		[]string{},
		map[string]interface{}{},
	)

	d.IgnoreKeys = append(d.IgnoreKeys, "^metadata")
	g.appendResource(d)

	return nil
}

// InitResources
func (g *DomainGenerator) InitResources() error {
	api, err := g.initializeAPI()
	if err != nil {
		return err
	}

	funcs := []func(*mgo.API, mgo.Domain) error{
		g.createDomainResource,
	}

	err = createResourcesPerDomain(api, funcs)
	if err != nil {
		return err
	}

	return nil
}

// createResourcesPerDomain
func createResourcesPerDomain(api *mgo.API, funcs []func(*mgo.API, mgo.Domain) error) error {
	page := 1
	pageSize := 250
	params := map[string]string{
		"pageSize": strconv.Itoa(pageSize),
		"page":     strconv.Itoa(page),
	}

	for {
		params["page"] = strconv.Itoa(page)

		domains, err := api.ListDomains(params)
		if err != nil {
			return err
		}

		for _, d := range domains {
			for _, f := range funcs {
				if err := f(api, d); err != nil {
					return err
				}
			}
		}
		if len(domains) < pageSize {
			break
		}
		page++
	}
	return nil
}

func maxConcurrentMyrasecRequests() int {
	limit := runtime.NumCPU() / 2
	if limit < 1 {
		return 1
	}
	return limit
}

// createResourcesPerSubDomain
func createResourcesPerSubDomain(api *mgo.API, funcs []func(*mgo.API, int, mgo.VHost) error, onDomainLevel bool) error {
	page := 1
	pageSize := 250
	params := map[string]string{
		"pageSize": strconv.Itoa(pageSize),
		"page":     strconv.Itoa(page),
	}

	limit := maxConcurrentMyrasecRequests()
	for {
		params["page"] = strconv.Itoa(page)

		domains, err := api.ListDomains(params)
		if err != nil {
			return err
		}

		group := errgroup.Group{}
		group.SetLimit(limit)
		for _, d := range domains {
			domain := d

			// try to load data for ALL-{domainId}.
			if onDomainLevel {
				for _, f := range funcs {
					createResource := f
					group.Go(func() error {
						if err := createResource(api, domain.ID, mgo.VHost{Label: fmt.Sprintf("ALL-%d.", domain.ID)}); err != nil {
							return fmt.Errorf("create domain-level resources for domain %d: %w", domain.ID, err)
						}
						return nil
					})
				}
			}
			group.Go(func() error {
				if err := createResourcesPerVHost(api, domain, funcs); err != nil {
					return fmt.Errorf("create subdomain resources for domain %d: %w", domain.ID, err)
				}
				return nil
			})
		}
		if err := group.Wait(); err != nil {
			return err
		}
		if len(domains) < pageSize {
			break
		}
		page++
	}
	return nil
}

// createResourcesPerVHost
func createResourcesPerVHost(api *mgo.API, domain mgo.Domain, funcs []func(*mgo.API, int, mgo.VHost) error) error {
	page := 1
	pageSize := 250
	params := map[string]string{
		"pageSize": strconv.Itoa(pageSize),
		"page":     strconv.Itoa(page),
	}

	limit := maxConcurrentMyrasecRequests()
	for {
		params["page"] = strconv.Itoa(page)

		vhosts, err := api.ListAllSubdomainsForDomain(domain.ID, params)
		if err != nil {
			return err
		}

		group := errgroup.Group{}
		group.SetLimit(limit)
		for _, v := range vhosts {
			vhost := v
			for _, f := range funcs {
				createResource := f
				group.Go(func() error {
					if err := createResource(api, domain.ID, vhost); err != nil {
						return fmt.Errorf("create subdomain resources for domain %d subdomain %q: %w", domain.ID, vhost.Label, err)
					}
					return nil
				})
			}
		}
		if err := group.Wait(); err != nil {
			return err
		}
		if len(vhosts) < pageSize {
			break
		}
		page++
	}
	return nil
}
