package main

import (
	"os"
	"regexp"
	sortPkg "sort"
	"strconv"

	"golang.org/x/exp/constraints"
)

func getEnvInt(key string, fallback int) (int, error) {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return fallback, nil
	}

	return strconv.Atoi(value)
}

func mustGetEnvInt(key string, fallback int) int {
	value, err := getEnvInt(key, fallback)
	if err != nil {
		panic(err)
	}
	return value
}

func getEnvString(key string, fallback string) (string, error) {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		value = fallback
	}

	return value, nil
}

func mustGetEnvString(key string, fallback string) string {
	value, err := getEnvString(key, fallback)
	if err != nil {
		panic(err)
	}
	return value
}

func getEnvStringSlice(key string, fallback string) ([]string, error) {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		value = fallback
	}

	return regexp.MustCompile(`\s*(,\s*)+`).Split(value, -1), nil
}

func mustGetEnvStringSlice(key string, fallback string) []string {
	value, err := getEnvStringSlice(key, fallback)
	if err != nil {
		panic(err)
	}
	return value
}

func getEnvIntSlice(key string, fallback string) ([]int, error) {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		value = fallback
	}

	var result []int
	for _, item := range regexp.MustCompile(`\s*(,\s*)+`).Split(value, -1) {
		if item == "" {
			continue
		}
		i, err := strconv.Atoi(item)
		if err != nil {
			return nil, err
		}
		result = append(result, i)
	}

	return result, nil
}

func mustGetEnvIntSlice(key string, fallback string) []int {
	value, err := getEnvIntSlice(key, fallback)
	if err != nil {
		panic(err)
	}
	return value
}

func must(args ...interface{}) []interface{} {
	argIsError := func(arg interface{}) bool {
		if arg == nil {
			return false
		}
		_, ok := arg.(error)
		return ok
	}

	var results []interface{}

	for _, arg := range args {
		if argIsError(arg) {
			panic(arg)
		} else {
			results = append(results, arg)
		}
	}

	return results
}

func notContains[T comparable](slice []T, value T) bool {
	for _, item := range slice {
		if item == value {
			return false
		}
	}
	return true
}

func unique[T comparable](slice []T) []T {
	seen := make(map[T]struct{})
	var result []T
	for _, item := range slice {
		if _, ok := seen[item]; !ok {
			seen[item] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}

// sort sorts a slice of any ordered type in ascending order
func sort[T constraints.Ordered](slice []T) []T {
	sortPkg.Slice(slice, func(i, j int) bool {
		return slice[i] < slice[j]
	})
	return slice
}

// sortDesc sorts a slice of any ordered type in descending order
func sortDesc[T constraints.Ordered](slice []T) []T {
	sortPkg.Slice(slice, func(i, j int) bool {
		return slice[i] > slice[j]
	})
	return slice
}

// sortBy sorts a slice of any type using a custom comparison function
func sortBy[T any](slice []T, less func(a, b T) bool) []T {
	sortPkg.Slice(slice, func(i, j int) bool {
		return less(slice[i], slice[j])
	})
	return slice
}
