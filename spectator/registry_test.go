package spectator

//func TestRegistry_Counter(t *testing.T) {
//	r := NewRegistry(config)
//	r.Counter("foo", nil).Increment()
//	if v := r.Counter("foo", nil).Count(); v != 1 {
//		t.Error("Counter needs to return a previously registered counter. Expected 1, got", v)
//	}
//}
//
//func TestRegistry_DistributionSummary(t *testing.T) {
//	r := NewRegistry(config)
//	r.DistributionSummary("ds", nil).Record(100)
//	if v := r.DistributionSummary("ds", nil).Count(); v != 1 {
//		t.Error("DistributionSummary needs to return a previously registered meter. Expected 1, got", v)
//	}
//	if v := r.DistributionSummary("ds", nil).TotalAmount(); v != 100 {
//		t.Error("Expected 100, Got", v)
//	}
//}
//
//func TestRegistry_Gauge(t *testing.T) {
//	r := NewRegistry(config)
//	r.Gauge("g", nil).Set(100)
//	if v := r.Gauge("g", nil).Get(); v != 100 {
//		t.Error("Gauge needs to return a previously registered meter. Expected 100, got", v)
//	}
//}
//
//func TestRegistry_Timer(t *testing.T) {
//	r := NewRegistry(config)
//	r.Timer("t", nil).Record(100)
//	if v := r.Timer("t", nil).Count(); v != 1 {
//		t.Error("Timer needs to return a previously registered meter. Expected 1, got", v)
//	}
//	if v := r.Timer("t", nil).TotalTime(); v != 100 {
//		t.Error("Expected 100, Got", v)
//	}
//}
