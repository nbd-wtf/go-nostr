package negentropy

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSimple(t *testing.T) {
	var err error
	var q []byte
	var n1 *Negentropy
	var n2 *Negentropy

	{
		n1, _ = NewNegentropy(NewVector(), 1<<16)
		n1.Insert(10, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
		n1.Insert(20, "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
		n1.Insert(30, "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc")
		n1.Insert(40, "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd")
		n1.Seal()

		q, err = n1.Initiate()
		if err != nil {
			t.Fatal(err)
			return
		}

		fmt.Println("n1:", q)
	}

	{
		n2, _ = NewNegentropy(NewVector(), 1<<16)
		n2.Insert(20, "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
		n2.Insert(30, "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc")
		n2.Insert(50, "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee")
		n2.Seal()

		q, err = n2.Reconcile(q)
		if err != nil {
			t.Fatal(err)
			return
		}
		fmt.Println("n2:", q)
	}

	{
		var have []string
		var need []string
		q, err = n1.ReconcileWithIDs(q, &have, &need)
		if err != nil {
			t.Fatal(err)
			return
		}
		fmt.Println("n1:", q)
		fmt.Println("have", have)
		fmt.Println("need", need)

		require.Equal(t, have, []string{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"})
		require.Equal(t, need, []string{"eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"})
	}
}
