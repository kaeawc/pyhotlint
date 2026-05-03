package oracle

import (
	"os/exec"
	"testing"
)

func TestStub_AlwaysUnknown(t *testing.T) {
	s := Stub{}
	if got := s.DeviceOf("model.weight"); got != Unknown {
		t.Errorf("DeviceOf: got %v, want Unknown", got)
	}
	if got := s.SubclassesNNModule("foo.Bar"); got != Unknown {
		t.Errorf("SubclassesNNModule: got %v, want Unknown", got)
	}
	if err := s.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestFake_ProgrammableResponses(t *testing.T) {
	f := NewFake()
	f.Devices["x"] = Result{Known: true, Value: "cuda:0"}
	f.NNModules["torchvision.models.resnet.ResNet"] = Result{Known: true, Value: "yes"}

	if got := f.DeviceOf("x"); got.Value != "cuda:0" || !got.Known {
		t.Errorf("DeviceOf(x): got %v", got)
	}
	if got := f.DeviceOf("y"); got != Unknown {
		t.Errorf("DeviceOf(y) should be Unknown, got %v", got)
	}
	if got := f.SubclassesNNModule("torchvision.models.resnet.ResNet"); got.Value != "yes" {
		t.Errorf("SubclassesNNModule: got %v", got)
	}
}

func TestSubprocess_RoundTrip(t *testing.T) {
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 not available")
	}
	s, err := Start(t.Context(), python)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer s.Close()

	// device_of always returns Unknown in this MVP — the helper does
	// not implement symbol resolution. We just verify the round trip.
	got := s.DeviceOf("anything")
	if got.Known {
		t.Fatalf("DeviceOf should be Unknown (not yet implemented), got %v", got)
	}

	// subclasses_nn_module against a non-existent qualname must
	// resolve to Unknown without crashing the subprocess.
	got = s.SubclassesNNModule("nonexistent.module.Foo")
	if got.Known {
		t.Fatalf("SubclassesNNModule on bogus name should be Unknown, got %v", got)
	}
}

func TestSubprocess_CloseIdempotent(t *testing.T) {
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 not available")
	}
	s, err := Start(t.Context(), python)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Errorf("first Close: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Errorf("second Close: %v", err)
	}
}

func TestSubprocess_CallAfterCloseErrors(t *testing.T) {
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 not available")
	}
	s, err := Start(t.Context(), python)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	s.Close()
	got := s.DeviceOf("anything")
	if got.Known {
		t.Fatalf("after Close, DeviceOf should be Unknown, got %v", got)
	}
}

func TestDiscoverPython_Fallback(t *testing.T) {
	// Empty root should fall through to PATH-based python3/python.
	got := DiscoverPython("")
	// On systems without any python it returns ""; we just verify
	// the function does not panic.
	_ = got
}
