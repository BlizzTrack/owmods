package helpers

import "net/url"

func ToURLQuery(values map[string]string) url.Values {
	query := url.Values{}

	for key, val := range values {
		query.Add(key, val)
	}

	return query
}