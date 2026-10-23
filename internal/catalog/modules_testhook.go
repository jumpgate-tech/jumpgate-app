package catalog

// SwapModulesForTest replaces the module registry until restore is called,
// for other packages' tests: the catalog registers no real module until each
// module's own task lands, and the routes, setup and status all read the
// registry through the catalog. The list is sorted and checked the way the
// real registry is at init, and a bad one panics.
func SwapModulesForTest(ms []Module) (restore func()) {
	ms = append([]Module(nil), ms...)
	sortModules(ms)
	if err := checkModules(ms); err != nil {
		panic(err)
	}
	old := modules
	modules = ms
	return func() { modules = old }
}
