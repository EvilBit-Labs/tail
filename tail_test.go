// Copyright (c) 2019 FOSS contributors of https://github.com/nxadm/tail
// Copyright (c) 2015 HPE Software Inc. All rights reserved.
// Copyright (c) 2013 ActiveState Software Inc. All rights reserved.

// TODO:
//  * repeat all the tests with Poll:true

package tail

import (
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/nxadm/tail/ratelimiter"
	"github.com/nxadm/tail/watch"
)

func ExampleTailFile() { //nolint:testableexamples // example is for documentation, output is non-deterministic
	// Keep tracking a file even when recreated.
	// /var/log/messages is typically continuously written and rotated daily.
	testFileName := "/var/log/messages"
	// ReOpen when truncated, Follow to wait for new input when EOL is reached
	tailedFile, err := TailFile(testFileName, Config{ReOpen: true, Follow: true})
	if err != nil {
		panic(err)
	}

	for line := range tailedFile.Lines {
		fmt.Println(line.Text)
	}
	// Prints all the lines in the logfile and keeps printing new input
}

func TestMain(m *testing.M) {
	// Use a smaller poll duration for faster test runs. Keep it below
	// 100ms (which value is used as common delays for tests)
	watch.POLL_DURATION = 5 * time.Millisecond
	os.Exit(m.Run())
}

func TestMustExist(t *testing.T) {
	tail, err := TailFile("/no/such/file", Config{Follow: true, MustExist: true})
	if err == nil {
		t.Error("MustExist:true is violated")
		_ = tail.Stop()
	}
	tail, err = TailFile("/no/such/file", Config{Follow: true, MustExist: false})
	if err != nil {
		t.Error("MustExist:false is violated")
	}
	_ = tail.Stop()
	_, err = TailFile("README.md", Config{Follow: true, MustExist: true})
	if err != nil {
		t.Error("MustExist:true on an existing file is violated")
	}
	tail.Cleanup()
}

func TestWaitsForFileToExist(t *testing.T) {
	tailTest, cleanup := NewTailTest(t, "waits-for-file-to-exist")
	defer cleanup()
	tail := tailTest.StartTail("test.txt", Config{})
	go tailTest.VerifyTailOutput(tail, []string{"hello", "world"}, false)

	<-time.After(100 * time.Millisecond)
	tailTest.CreateFile("test.txt", "hello\nworld\n")
	tailTest.Cleanup(tail, true)
}

func TestWaitsForFileToExistRelativePath(t *testing.T) {
	tailTest, cleanup := NewTailTest(t, "waits-for-file-to-exist-relative")
	defer cleanup()

	oldWD, err := os.Getwd()
	if err != nil {
		tailTest.Fatal(err)
	}
	if err = os.Chdir(tailTest.path); err != nil {
		tailTest.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(oldWD); err != nil {
			tailTest.Logf("failed to restore working directory: %v", err)
		}
	}()

	tail, err := TailFile("test.txt", Config{})
	if err != nil {
		tailTest.Fatal(err)
	}

	go tailTest.VerifyTailOutput(tail, []string{"hello", "world"}, false)

	<-time.After(100 * time.Millisecond)
	if err := os.WriteFile("test.txt", []byte("hello\nworld\n"), 0o600); err != nil {
		tailTest.Fatal(err)
	}
	tailTest.Cleanup(tail, true)
}

func TestStop(t *testing.T) {
	tail, err := TailFile("_no_such_file", Config{Follow: true, MustExist: false})
	if err != nil {
		t.Error("MustExist:false is violated")
	}
	if tail.Stop() != nil {
		t.Error("Should be stoped successfully")
	}
	tail.Cleanup()
}

func TestStopNonEmptyFile(t *testing.T) {
	tailTest, cleanup := NewTailTest(t, "maxlinesize")
	defer cleanup()
	tailTest.CreateFile("test.txt", "hello\nthere\nworld\n")
	tail := tailTest.StartTail("test.txt", Config{})
	_ = tail.Stop()
	tail.Cleanup()
	// success here is if it doesn't panic.
}

func TestStopAtEOF(t *testing.T) {
	tailTest, cleanup := NewTailTest(t, "maxlinesize")
	defer cleanup()
	tailTest.CreateFile("test.txt", "hello\nthere\nworld\n")
	tail := tailTest.StartTail("test.txt", Config{Follow: true, Location: nil})

	// read "hello"
	line := <-tail.Lines
	if line.Text != "hello" {
		t.Errorf("Expected to get 'hello', got '%s' instead", line.Text)
	}

	if line.Num != 1 {
		t.Errorf("Expected to get 1, got %d instead", line.Num)
	}

	tailTest.VerifyTailOutput(tail, []string{"there", "world"}, false)
	_ = tail.StopAtEOF()
	tailTest.Cleanup(tail, true)
}

func TestMaxLineSizeFollow(t *testing.T) {
	// As last file line does not end with newline, it will not be present in tail's output
	maxLineSize(t, true, "hello\nworld\nfin\nhe", []string{"hel", "lo", "wor", "ld", "fin", "he"})
}

func TestMaxLineSizeNoFollow(t *testing.T) {
	maxLineSize(t, false, "hello\nworld\nfin\nhe", []string{"hel", "lo", "wor", "ld", "fin", "he"})
}

func TestOver4096ByteLine(t *testing.T) {
	tailTest, cleanup := NewTailTest(t, "Over4096ByteLine")
	defer cleanup()
	testString := strings.Repeat("a", 4097)
	tailTest.CreateFile("test.txt", "test\n"+testString+"\nhello\nworld\n")
	tail := tailTest.StartTail("test.txt", Config{Follow: true, Location: nil})
	go tailTest.VerifyTailOutput(tail, []string{"test", testString, "hello", "world"}, false)

	// Delete after a reasonable delay, to give tail sufficient time
	// to read all lines.
	<-time.After(100 * time.Millisecond)
	tailTest.RemoveFile("test.txt")
	tailTest.Cleanup(tail, true)
}

func TestOver4096ByteLineWithSetMaxLineSize(t *testing.T) {
	tailTest, cleanup := NewTailTest(t, "Over4096ByteLineMaxLineSize")
	defer cleanup()
	testString := strings.Repeat("a", 4097)
	tailTest.CreateFile("test.txt", "test\n"+testString+"\nhello\nworld\n")
	tail := tailTest.StartTail("test.txt", Config{Follow: true, Location: nil, MaxLineSize: 4097})
	go tailTest.VerifyTailOutput(tail, []string{"test", testString, "hello", "world"}, false)

	// Delete after a reasonable delay, to give tail sufficient time
	// to read all lines.
	<-time.After(100 * time.Millisecond)
	tailTest.RemoveFile("test.txt")
	tailTest.Cleanup(tail, true)
}

func TestReOpenWithCursor(t *testing.T) {
	delay := 300 * time.Millisecond // account for POLL_DURATION
	tailTest, cleanup := NewTailTest(t, "reopen-cursor")
	defer cleanup()
	tailTest.CreateFile("test.txt", "hello\nworld\n")
	tail := tailTest.StartTail(
		"test.txt",
		Config{Follow: true, ReOpen: true, Poll: true})
	content := []string{"hello", "world", "more", "data", "endofworld"}
	go tailTest.VerifyTailOutputUsingCursor(tail, content, false)

	// deletion must trigger reopen
	<-time.After(delay)
	tailTest.RemoveFile("test.txt")
	<-time.After(delay)
	tailTest.CreateFile("test.txt", "hello\nworld\nmore\ndata\n")

	// rename must trigger reopen
	<-time.After(delay)
	tailTest.RenameFile("test.txt", "test.txt.rotated")
	<-time.After(delay)
	tailTest.CreateFile("test.txt", "hello\nworld\nmore\ndata\nendofworld\n")

	// Delete after a reasonable delay, to give tail sufficient time
	// to read all lines.
	<-time.After(delay)
	tailTest.RemoveFile("test.txt")
	<-time.After(delay)

	// Do not bother with stopping as it could kill the tomb during
	// the reading of data written above. Timings can vary based on
	// test environment.
	tailTest.Cleanup(tail, false)
}

func TestLocationFull(t *testing.T) {
	tailTest, cleanup := NewTailTest(t, "location-full")
	defer cleanup()
	tailTest.CreateFile("test.txt", "hello\nworld\n")
	tail := tailTest.StartTail("test.txt", Config{Follow: true, Location: nil})
	go tailTest.VerifyTailOutput(tail, []string{"hello", "world"}, false)

	// Delete after a reasonable delay, to give tail sufficient time
	// to read all lines.
	<-time.After(100 * time.Millisecond)
	tailTest.RemoveFile("test.txt")
	tailTest.Cleanup(tail, true)
}

func TestLocationFullDontFollow(t *testing.T) {
	tailTest, cleanup := NewTailTest(t, "location-full-dontfollow")
	defer cleanup()
	tailTest.CreateFile("test.txt", "hello\nworld\n")
	tail := tailTest.StartTail("test.txt", Config{Follow: false, Location: nil})
	go tailTest.VerifyTailOutput(tail, []string{"hello", "world"}, false)

	// Add more data only after reasonable delay.
	<-time.After(100 * time.Millisecond)
	tailTest.AppendFile("test.txt", "more\ndata\n")
	<-time.After(100 * time.Millisecond)

	tailTest.Cleanup(tail, true)
}

func TestLocationEnd(t *testing.T) {
	tailTest, cleanup := NewTailTest(t, "location-end")
	defer cleanup()
	tailTest.CreateFile("test.txt", "hello\nworld\n")
	tail := tailTest.StartTail("test.txt", Config{Follow: true, Location: &SeekInfo{0, io.SeekEnd}})
	go tailTest.VerifyTailOutput(tail, []string{"more", "data"}, false)

	<-time.After(100 * time.Millisecond)
	tailTest.AppendFile("test.txt", "more\ndata\n")

	// Delete after a reasonable delay, to give tail sufficient time
	// to read all lines.
	<-time.After(100 * time.Millisecond)
	tailTest.RemoveFile("test.txt")
	tailTest.Cleanup(tail, true)
}

func TestLocationMiddle(t *testing.T) {
	// Test reading from middle.
	tailTest, cleanup := NewTailTest(t, "location-middle")
	defer cleanup()
	tailTest.CreateFile("test.txt", "hello\nworld\n")
	tail := tailTest.StartTail("test.txt", Config{Follow: true, Location: &SeekInfo{-6, io.SeekEnd}})
	go tailTest.VerifyTailOutput(tail, []string{"world", "more", "data"}, false)

	<-time.After(100 * time.Millisecond)
	tailTest.AppendFile("test.txt", "more\ndata\n")

	// Delete after a reasonable delay, to give tail sufficient time
	// to read all lines.
	<-time.After(100 * time.Millisecond)
	tailTest.RemoveFile("test.txt")
	tailTest.Cleanup(tail, true)
}

// The use of polling file watcher could affect file rotation
// (detected via renames), so test these explicitly.

func TestReOpenInotify(t *testing.T) {
	reOpen(t, false)
}

func TestReOpenPolling(t *testing.T) {
	reOpen(t, true)
}

// The use of polling file watcher could affect file rotation
// (detected via renames), so test these explicitly.

func TestReSeekInotify(t *testing.T) {
	reSeek(t, false)
}

func TestReSeekPolling(t *testing.T) {
	reSeek(t, true)
}

func TestReSeekWithCursor(t *testing.T) {
	tailTest, cleanup := NewTailTest(t, "reseek-cursor")
	defer cleanup()
	tailTest.CreateFile("test.txt", "a really long string goes here\nhello\nworld\n")
	tail := tailTest.StartTail(
		"test.txt",
		Config{Follow: true, ReOpen: false, Poll: false})

	go tailTest.VerifyTailOutputUsingCursor(tail, []string{
		"a really long string goes here", "hello", "world", "but", "not", "me",
	}, false)

	// truncate now
	<-time.After(100 * time.Millisecond)
	tailTest.TruncateFile("test.txt", "skip\nme\nplease\nbut\nnot\nme\n")

	// Delete after a reasonable delay, to give tail sufficient time
	// to read all lines.
	<-time.After(100 * time.Millisecond)
	tailTest.RemoveFile("test.txt")

	// Do not bother with stopping as it could kill the tomb during
	// the reading of data written above. Timings can vary based on
	// test environment.
	tailTest.Cleanup(tail, false)
}

func TestRateLimiting(t *testing.T) {
	tailTest, cleanup := NewTailTest(t, "rate-limiting")
	defer cleanup()
	tailTest.CreateFile("test.txt", "hello\nworld\nagain\nextra\n")
	config := Config{
		Follow:      true,
		RateLimiter: ratelimiter.NewLeakyBucket(2, time.Second),
	}
	leakybucketFull := "Too much log activity; waiting a second before resuming tailing"
	tail := tailTest.StartTail("test.txt", config)

	// TODO: also verify that tail resumes after the cooloff period.
	go tailTest.VerifyTailOutput(tail, []string{
		"hello", "world", "again",
		leakybucketFull,
		"more", "data",
		leakybucketFull,
	}, false)

	// Add more data only after reasonable delay.
	<-time.After(1200 * time.Millisecond)
	tailTest.AppendFile("test.txt", "more\ndata\n")

	// Delete after a reasonable delay, to give tail sufficient time
	// to read all lines.
	<-time.After(100 * time.Millisecond)
	tailTest.RemoveFile("test.txt")

	tailTest.Cleanup(tail, true)
}

func TestTell(t *testing.T) {
	tailTest, cleanup := NewTailTest(t, "tell-position")
	defer cleanup()
	tailTest.CreateFile("test.txt", "hello\nworld\nagain\nmore\n")
	config := Config{
		Follow:   false,
		Location: &SeekInfo{0, io.SeekStart},
	}
	tail := tailTest.StartTail("test.txt", config)
	// read one line
	line := <-tail.Lines
	if line.Num != 1 {
		tailTest.Errorf("expected line to have number 1 but got %d", line.Num)
	}
	offset, err := tail.Tell()
	if err != nil {
		tailTest.Errorf("Tell return error: %s", err.Error())
	}
	_ = tail.Stop()

	config = Config{
		Follow:   false,
		Location: &SeekInfo{offset, io.SeekStart},
	}
	tail = tailTest.StartTail("test.txt", config)
	for l := range tail.Lines {
		// it may readed one line in the chan(tail.Lines),
		// so it may lost one line.
		if l.Text != "world" && l.Text != "again" {
			tailTest.Fatalf("mismatch; expected world or again, but got %s",
				l.Text)
		}
		break //nolint:staticcheck // SA4004: intentionally reading only the first line
	}
	tailTest.RemoveFile("test.txt")
	_ = tail.Stop()
	tail.Cleanup()
}

func TestBlockUntilExists(t *testing.T) {
	tailTest, cleanup := NewTailTest(t, "block-until-file-exists")
	defer cleanup()
	config := Config{
		Follow: true,
	}
	tail := tailTest.StartTail("test.txt", config)
	go func() {
		time.Sleep(100 * time.Millisecond)
		tailTest.CreateFile("test.txt", "hello world\n")
	}()
	for l := range tail.Lines {
		if l.Text != "hello world" {
			tailTest.Fatalf("mismatch; expected hello world, but got %s",
				l.Text)
		}
		break //nolint:staticcheck // SA4004: intentionally reading only the first line
	}
	tailTest.RemoveFile("test.txt")
	_ = tail.Stop()
	tail.Cleanup()
}

func maxLineSize(t *testing.T, follow bool, fileContent string, expected []string) {
	t.Helper()
	tailTest, cleanup := NewTailTest(t, "maxlinesize")
	defer cleanup()
	tailTest.CreateFile("test.txt", fileContent)
	tail := tailTest.StartTail("test.txt", Config{Follow: follow, Location: nil, MaxLineSize: 3})
	go tailTest.VerifyTailOutput(tail, expected, false)

	// Delete after a reasonable delay, to give tail sufficient time
	// to read all lines.
	<-time.After(100 * time.Millisecond)
	tailTest.RemoveFile("test.txt")
	tailTest.Cleanup(tail, true)
}

func reOpen(t *testing.T, poll bool) {
	t.Helper()
	var name string
	var delay time.Duration
	if poll {
		name = "reopen-polling"
		delay = 300 * time.Millisecond // account for POLL_DURATION
	} else {
		name = "reopen-inotify"
		delay = 100 * time.Millisecond
	}
	tailTest, cleanup := NewTailTest(t, name)
	defer cleanup()
	tailTest.CreateFile("test.txt", "hello\nworld\n")
	tail := tailTest.StartTail(
		"test.txt",
		Config{Follow: true, ReOpen: true, Poll: poll})
	content := []string{"hello", "world", "more", "data", "endofworld"}
	go tailTest.VerifyTailOutput(tail, content, false)

	if poll {
		// deletion must trigger reopen
		<-time.After(delay)
		tailTest.RemoveFile("test.txt")
		<-time.After(delay)
		tailTest.CreateFile("test.txt", "more\ndata\n")
	} else {
		// In inotify mode, fsnotify is currently unable to deliver notifications
		// about deletion of open files, so we are not testing file deletion.
		// (see https://github.com/fsnotify/fsnotify/issues/194 for details).
		<-time.After(delay)
		tailTest.AppendToFile("test.txt", "more\ndata\n")
	}

	// rename must trigger reopen
	<-time.After(delay)
	tailTest.RenameFile("test.txt", "test.txt.rotated")
	<-time.After(delay)
	tailTest.CreateFile("test.txt", "endofworld\n")

	// Delete after a reasonable delay, to give tail sufficient time
	// to read all lines.
	<-time.After(delay)
	tailTest.RemoveFile("test.txt")
	<-time.After(delay)

	// Do not bother with stopping as it could kill the tomb during
	// the reading of data written above. Timings can vary based on
	// test environment.
	tailTest.Cleanup(tail, false)
}

func TestInotify_WaitForCreateThenMove(t *testing.T) {
	tailTest, cleanup := NewTailTest(t, "wait-for-create-then-reopen")
	defer cleanup()
	_ = os.Remove(tailTest.path + "/test.txt") // Make sure the file does NOT exist.

	tail := tailTest.StartTail(
		"test.txt",
		Config{Follow: true, ReOpen: true, Poll: false})

	content := []string{"hello", "world", "endofworld"}
	go tailTest.VerifyTailOutput(tail, content, false)

	time.Sleep(50 * time.Millisecond)
	tailTest.CreateFile("test.txt", "hello\nworld\n")
	time.Sleep(50 * time.Millisecond)
	tailTest.RenameFile("test.txt", "test.txt.rotated")
	time.Sleep(50 * time.Millisecond)
	tailTest.CreateFile("test.txt", "endofworld\n")
	time.Sleep(50 * time.Millisecond)
	tailTest.RemoveFile("test.txt.rotated")
	tailTest.RemoveFile("test.txt")

	// Do not bother with stopping as it could kill the tomb during
	// the reading of data written above. Timings can vary based on
	// test environment.
	tailTest.Cleanup(tail, false)
}

func TestIncompleteLines(t *testing.T) {
	tailTest, cleanup := NewTailTest(t, "incomplete-lines")
	defer cleanup()
	filename := "test.txt" //nolint:goconst // test filename, not worth extracting
	config := Config{
		Follow:        true,
		CompleteLines: true,
	}
	tail := tailTest.StartTail(filename, config)
	go func() {
		time.Sleep(100 * time.Millisecond)
		tailTest.CreateFile(filename, "hello world\n")
		time.Sleep(100 * time.Millisecond)
		// here we intentially write a partial line to see if `Tail` contains
		// information that it's incomplete
		tailTest.AppendFile(filename, "hello")
		time.Sleep(100 * time.Millisecond)
		tailTest.AppendFile(filename, " again\n")
	}()

	lines := []string{"hello world", "hello again"}

	tailTest.ReadLines(tail, lines, false)

	tailTest.RemoveFile(filename)
	_ = tail.Stop()
	tail.Cleanup()
}

func TestIncompleteLongLines(t *testing.T) {
	tailTest, cleanup := NewTailTest(t, "incomplete-lines-long")
	defer cleanup()
	filename := "test.txt"
	config := Config{
		Follow:        true,
		MaxLineSize:   3,
		CompleteLines: true,
	}
	tail := tailTest.StartTail(filename, config)
	go func() {
		time.Sleep(100 * time.Millisecond)
		tailTest.CreateFile(filename, "hello world\n")
		time.Sleep(100 * time.Millisecond)
		tailTest.AppendFile(filename, "hello")
		time.Sleep(100 * time.Millisecond)
		tailTest.AppendFile(filename, "again\n")
	}()

	lines := []string{"hel", "lo ", "wor", "ld", "hel", "loa", "gai", "n"}

	tailTest.ReadLines(tail, lines, false)

	tailTest.RemoveFile(filename)
	_ = tail.Stop()
	tail.Cleanup()
}

func TestIncompleteLinesWithReopens(t *testing.T) {
	tailTest, cleanup := NewTailTest(t, "incomplete-lines-reopens")
	defer cleanup()
	filename := "test.txt"
	config := Config{
		Follow:        true,
		CompleteLines: true,
	}
	tail := tailTest.StartTail(filename, config)
	go func() {
		time.Sleep(100 * time.Millisecond)
		tailTest.CreateFile(filename, "hello world\nhi")
		time.Sleep(100 * time.Millisecond)
		tailTest.TruncateFile(filename, "rewriting\n")
	}()

	// not that the "hi" gets lost, because it was never a complete line
	lines := []string{"hello world", "rewriting"}

	tailTest.ReadLines(tail, lines, false)

	tailTest.RemoveFile(filename)
	_ = tail.Stop()
	tail.Cleanup()
}

func TestIncompleteLinesWithoutFollow(t *testing.T) {
	tailTest, cleanup := NewTailTest(t, "incomplete-lines-no-follow")
	defer cleanup()
	filename := "test.txt"
	config := Config{
		Follow:        false,
		CompleteLines: true,
	}
	tail := tailTest.StartTail(filename, config)
	go func() {
		time.Sleep(100 * time.Millisecond)
		// intentionally missing a newline at the end
		tailTest.CreateFile(filename, "foo\nbar\nbaz")
	}()

	lines := []string{"foo", "bar", "baz"}

	tailTest.VerifyTailOutput(tail, lines, true)

	tailTest.RemoveFile(filename)
	_ = tail.Stop()
	tail.Cleanup()
}

func reSeek(t *testing.T, poll bool) {
	t.Helper()
	var name string
	if poll {
		name = "reseek-polling"
	} else {
		name = "reseek-inotify"
	}
	tailTest, cleanup := NewTailTest(t, name)
	defer cleanup()
	tailTest.CreateFile("test.txt", "a really long string goes here\nhello\nworld\n")
	tail := tailTest.StartTail(
		"test.txt",
		Config{Follow: true, ReOpen: false, Poll: poll})

	go tailTest.VerifyTailOutput(tail, []string{
		"a really long string goes here", "hello", "world", "h311o", "w0r1d", "endofworld",
	}, false)

	// truncate now
	<-time.After(100 * time.Millisecond)
	tailTest.TruncateFile("test.txt", "h311o\nw0r1d\nendofworld\n")

	// Delete after a reasonable delay, to give tail sufficient time
	// to read all lines.
	<-time.After(100 * time.Millisecond)
	tailTest.RemoveFile("test.txt")

	// Do not bother with stopping as it could kill the tomb during
	// the reading of data written above. Timings can vary based on
	// test environment.
	tailTest.Cleanup(tail, false)
}

// Test library

type TailTest struct {
	*testing.T

	Name string
	path string
	done chan struct{}
}

func NewTailTest(
	t *testing.T,
	name string,
) (TailTest, func()) { //nolint:gocritic // unnamedResult: names conflict with nonamedreturns linter
	t.Helper()
	testdir := t.TempDir()

	return TailTest{t, name, testdir, make(chan struct{})}, func() {}
}

func (t TailTest) CreateFile(name, contents string) {
	err := os.WriteFile(t.path+"/"+name, []byte(contents), 0o600)
	if err != nil {
		t.Fatal(err)
	}
}

func (t TailTest) AppendToFile(name, contents string) {
	err := os.WriteFile(t.path+"/"+name, []byte(contents), 0o600|os.ModeAppend)
	if err != nil {
		t.Fatal(err)
	}
}

func (t TailTest) RemoveFile(name string) {
	err := os.Remove(t.path + "/" + name)
	if err != nil {
		t.Fatal(err)
	}
}

func (t TailTest) RenameFile(oldname, newname string) {
	oldname = t.path + "/" + oldname
	newname = t.path + "/" + newname
	err := os.Rename(oldname, newname)
	if err != nil {
		t.Fatal(err)
	}
}

func (t TailTest) AppendFile(name, contents string) {
	f, err := os.OpenFile(t.path+"/"+name, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	_, err = f.WriteString(contents)
	if err != nil {
		t.Fatal(err)
	}
}

func (t TailTest) TruncateFile(name, contents string) {
	f, err := os.OpenFile(t.path+"/"+name, os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	_, err = f.WriteString(contents)
	if err != nil {
		t.Fatal(err)
	}
}

func (t TailTest) StartTail(name string, config Config) *Tail {
	tail, err := TailFile(t.path+"/"+name, config)
	if err != nil {
		t.Fatal(err)
	}
	return tail
}

func (t TailTest) VerifyTailOutput(tail *Tail, lines []string, expectEOF bool) {
	defer close(t.done)
	t.ReadLines(tail, lines, false)
	// It is important to do this if only EOF is expected
	// otherwise we could block on <-tail.Lines
	if expectEOF {
		line, ok := <-tail.Lines
		if ok {
			t.Fatalf("more content from tail: %+v", line)
		}
	}
}

func (t TailTest) VerifyTailOutputUsingCursor(tail *Tail, lines []string, expectEOF bool) {
	defer close(t.done)
	t.ReadLines(tail, lines, true)
	// It is important to do this if only EOF is expected
	// otherwise we could block on <-tail.Lines
	if expectEOF {
		line, ok := <-tail.Lines
		if ok {
			t.Fatalf("more content from tail: %+v", line)
		}
	}
}

func (t TailTest) ReadLines(tail *Tail, lines []string, useCursor bool) {
	cursor := 1

	for _, line := range lines {
		for {
			tailedLine, ok := <-tail.Lines
			if !ok {
				// tail.Lines is closed and empty.
				err := tail.Err()
				if err != nil {
					t.Fatalf("tail ended with error: %v", err)
				}
				t.Fatalf("tail ended early; expecting more: %v", lines[cursor:])
			}
			if tailedLine == nil {
				t.Fatalf("tail.Lines returned nil; not possible")
			}

			if useCursor && tailedLine.Num < cursor {
				// skip lines up until cursor
				continue
			}

			// Note: not checking .Err as the `lines` argument is designed
			// to match error strings as well.
			if tailedLine.Text != line {
				t.Fatalf(
					"unexpected line/err from tail: "+
						"expecting <<%s>>>, but got <<<%s>>>",
					line, tailedLine.Text)
			}

			cursor++
			break
		}
	}
}

func (t TailTest) Cleanup(tail *Tail, stop bool) {
	<-t.done
	if stop {
		_ = tail.Stop()
	}
	tail.Cleanup()
}
