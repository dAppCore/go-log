package main

import core "dappco.re/go"

func TestMain_run_Good(t *core.T) {
	r := run()
	core.AssertTrue(t, r.OK)
	core.AssertNil(t, r.Value)
}

func TestMain_containsAll_Bad(t *core.T) {
	r := containsAll("present", []string{"missing"})
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "output missing")
}

func TestMain_lineCount_Ugly(t *core.T) {
	empty := lineCount("")
	two := lineCount("first\nsecond\n")
	core.AssertEqual(t, 0, empty)
	core.AssertEqual(t, 2, two)
}
