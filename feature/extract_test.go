package feature_test

import (
	"reflect"
	"testing"

	"github.com/liamlmy/OpsFeature/feature"
	_ "github.com/liamlmy/OpsFeature/operator"
)

// Lock down FID encoding, operator order, padding, and state reuse before refactoring.
func TestExtractionPipeline(t *testing.T) {
	const conf = `
name=price;class=Direct;slot_id=1;depend=price;args=1,100,double
name=slotted;class=Direct;slot_id=2;depend=price;args=2,100,double
name=bits;class=Direct;slot_id=3;depend=price;args=3,1,double
name=bucket;class=Bucket;slot_id=4;depend=price;args=1,1,abslisan,1,2,3
name=sequence;class=Seq;slot_id=5;depend=history;args=1,1,32768,32770,3,9
name=embedding;class=Seq_emb;slot_id=6;depend=vector;args=3,1,32771,32772,2,2,0
name=difference;class=Arithmetic;slot_id=7;depend=price,zero;args=minus,3,1
name=scaled;class=Scale;slot_id=8;depend=zero;args=3,1,log
`
	extractor, err := feature.NewFromText(conf)
	if err != nil {
		t.Fatal(err)
	}
	state := feature.NewState()
	var out []feature.Fid
	for range 2 {
		state.Reset()
		out, err = extractor.Extract(&feature.Input{Score: map[string]interface{}{
			"price": "1.5", "zero": "0", "history": "10_20", "vector": "1",
		}}, state, out[:0])
		if err != nil {
			t.Fatal(err)
		}
		want := []feature.Fid{
			{Slot: 1, Val: 150}, {Slot: 2, Val: 562949953421462},
			{Slot: 3, Val: 1069547520}, {Slot: 4, Val: 1},
			{Slot: 32768, Val: 10}, {Slot: 32769, Val: 20}, {Slot: 32770, Val: 9},
			{Slot: 32771, Val: 1065353216}, {Slot: 32772, Val: 0},
			{Slot: 7, Val: 1069547520}, {Slot: 8, Val: 0},
		}
		if !reflect.DeepEqual(out, want) {
			t.Fatalf("FID output = %v, want %v", out, want)
		}
	}
}

func TestOutputFormat(t *testing.T) {
	fields, err := feature.ParseFeatureDataPerLineStrict("trace-1\t0\t1\r", []string{"traceid", "negative", "positive"})
	if err != nil {
		t.Fatal(err)
	}
	output, err := feature.OutputPerLineOp(fields, []string{"negative", "positive"}, []string{"traceid"}, []feature.Fid{{Slot: 10, Val: 42}})
	if err != nil {
		t.Fatal(err)
	}
	if want := "1 10:42 #trace-1\t"; output != want {
		t.Fatalf("output = %q, want %q", output, want)
	}
}
