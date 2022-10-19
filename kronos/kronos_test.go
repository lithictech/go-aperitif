package kronos_test

import (
	"fmt"
	"github.com/lithictech/go-aperitif/kronos"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"math/rand"
	"time"
)

var _ = Describe("kronos.TMin/TMax", func() {
	Describe("TMin", func() {
		t1 := time.Now()
		t2 := t1.Add(time.Hour)

		It("selects the lesser date", func() {
			Expect(kronos.TMin(t1, t2)).To(BeIdenticalTo(t1))
		})
		It("selects the greater date", func() {
			Expect(kronos.TMin(t1, t2)).To(BeIdenticalTo(t2))
		})
	})
})

var _ = Describe("kronos.Between", func() {
	It("returns a slice of times between start and end", func() {
		start := time.Now()
		end := start.Add(5000 * time.Millisecond)
		bt := kronos.Between(start, end, 1100*time.Millisecond)
		Expect(bt).To(HaveLen(5))
		Expect(bt[0]).To(Equal(start))
		Expect(bt[1]).To(Equal(start.Add(1100 * time.Millisecond)))
		Expect(bt[4]).To(Equal(start.Add(4400 * time.Millisecond)))
	})

	It("is inclusive of start and end", func() {
		start := time.Now()
		end := start.Add(5000 * time.Millisecond)
		bt := kronos.Between(start, end, 1000*time.Millisecond)
		Expect(bt).To(HaveLen(6))
	})

	It("contains a single item if start is equal to end", func() {
		start := time.Now()
		Expect(kronos.Between(start, start, time.Hour)).To(HaveLen(1))
	})

	It("contains only start if end is less than interval from start", func() {
		start := time.Now()
		Expect(kronos.Between(start, start.Add(59*time.Minute), time.Hour)).To(Equal([]time.Time{start}))
	})

	It("is empty if end is after start", func() {
		start := time.Now()
		Expect(kronos.Between(start, start.Add(-20*time.Hour), time.Hour)).To(BeEmpty())
	})

	It("pre-allocates a slice of the correct size", func() {
		start := time.Now()
		bt := kronos.Between(start, start.Add(5000*time.Millisecond), 1100*time.Millisecond)
		Expect(bt).To(HaveLen(5))
		Expect(bt).To(HaveCap(5))
	})
})

var _ = Describe("kronos.RollMonths", func() {
	date := func(y, m, d int) time.Time {
		return time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
	}

	ymd := func(t time.Time) []int {
		return []int{t.Year(), int(t.Month()), t.Day()}
	}

	entry := func(arg time.Time, offset int, expected time.Time, note string) TableEntry {
		desc := fmt.Sprintf("%d/%d + %d => %d/%d (%s)",
			arg.Month(), arg.Day(), offset, expected.Month(), expected.Day(), note)
		return Entry(desc, arg, offset, expected)
	}

	DescribeTable("rolls months",
		func(arg time.Time, offset int, expected time.Time) {
			actual := kronos.RollMonth(arg, offset)
			Expect(ymd(actual)).To(Equal(ymd(expected)))
		},
		entry(date(2016, 8, 30), 1, date(2016, 9, 30), "forward a month"),
		entry(date(2016, 3, 31), 1, date(2016, 4, 30), "April has fewer days than March"),
		entry(date(2016, 3, 31), 2, date(2016, 5, 31), "through April, with fewer days"),
		entry(date(2016, 3, 31), 12, date(2017, 3, 31), "forward 12 months"),
		entry(date(2016, 12, 31), 1, date(2017, 1, 31), "forward over year boundary"),
		entry(date(2016, 1, 31), 1, date(2016, 2, 29), "forward into a shorter month"),
		entry(date(2016, 1, 31), 2, date(2016, 3, 31), "forward over a shorter month"),

		entry(date(2016, 9, 30), -1, date(2016, 8, 30), "back a month"),
		entry(date(2016, 3, 31), -1, date(2016, 2, 29), "leap feb has fewer days than march"),
		entry(date(2015, 3, 31), -1, date(2015, 2, 28), "nonleap feb has fewer days than march"),
		entry(date(2016, 3, 31), -2, date(2016, 1, 31), "through feb with fewer days"),
		entry(date(2016, 3, 31), -3, date(2015, 12, 31), "over year boundary"),
		entry(date(2016, 3, 31), -12, date(2015, 3, 31), "back 12 months"),
	)
})

var _ = Describe("kronos.DaysInMonth", func() {
	DescribeTable("returns the number of days in the month described by the given time",
		func(y, m, expected int) {
			t := time.Date(y, time.Month(m), rand.Intn(25)+1, 0, 0, 0, 0, time.UTC)
			Expect(kronos.DaysInMonth(t)).To(Equal(expected))
		},
		Entry("Jan has 31 days", 2015, 1, 31),
		Entry("Dec has 31 days", 2015, 12, 31),
		Entry("June has 30 days", 2015, 6, 30),
		Entry("Jan has 31 days", 2015, 1, 31),
		Entry("March has 31 days", 2015, 3, 31),
		Entry("Regular Feb has 28 days", 2015, 2, 28),
		Entry("Leap year Feb has 29 days", 2016, 2, 29),
	)
})

func ExampleBetween() {
	sta, _ := time.Parse(time.RFC3339, "2006-01-02T15:04:05Z")
	end, _ := time.Parse(time.RFC3339, "2006-01-02T15:05:05Z")
	bt := kronos.Between(sta, end, 28*time.Second)
	fmt.Println("Len:", len(bt))
	for _, t := range bt {
		fmt.Println(t.Format("15:04:05"))
	}
	// Output:
	// Len: 3
	// 15:04:05
	// 15:04:33
	// 15:05:01
}

var _ = Describe("kronos.BetweenDates", func() {
	It("returns a slice of times between start and end", func() {
		start := time.Now()
		end := start.AddDate(0, 0, 5)
		bt := kronos.BetweenDates(start, end, 0, 0, 2)
		Expect(bt).To(HaveLen(3))
		Expect(bt[0]).To(Equal(start))
		Expect(bt[1]).To(Equal(start.AddDate(0, 0, 2)))
		Expect(bt[2]).To(Equal(start.AddDate(0, 0, 4)))
	})

	It("is inclusive of start and end", func() {
		start := time.Now()
		end := start.AddDate(0, 0, 2)
		bt := kronos.BetweenDates(start, end, 0, 0, 2)
		Expect(bt).To(HaveLen(2))
		Expect(bt[0]).To(Equal(start))
		Expect(bt[1]).To(Equal(end))
	})

	It("contains a single item if start is equal to end", func() {
		start := time.Now()
		Expect(kronos.BetweenDates(start, start, 1, 1, 1)).To(HaveLen(1))
	})

	It("contains only start if end is less than interval from start", func() {
		start := time.Now()
		end := start.AddDate(0, 0, 5)
		Expect(kronos.BetweenDates(start, end, 0, 1, 0)).To(Equal([]time.Time{start}))
	})

	It("is empty if end is after start", func() {
		start := time.Now()
		Expect(kronos.BetweenDates(start, start.Add(time.Hour*24*60*-1), 0, 0, 1)).To(BeEmpty())
	})

	It("pre-allocates a slice of the correct size", func() {
		start := time.Now()
		bt := kronos.BetweenDates(start, start.AddDate(5, 0, 0), 1, 1, 1)
		Expect(bt).To(HaveLen(5))
		Expect(bt).To(HaveCap(5))
	})
})

func ExampleBetweenDates() {
	start, _ := time.Parse("2006-01-02", "2012-11-22")
	end, _ := time.Parse("2006-01-02", "2015-03-06")
	bt := kronos.BetweenDates(start, end, 1, 0, 0)
	fmt.Println("Len:", len(bt))
	for _, t := range bt {
		fmt.Println(t.Format("2006-01-02"))
	}
	// Output:
	// Len: 3
	// 2012-11-22
	// 2013-11-22
	// 2014-11-22
}
