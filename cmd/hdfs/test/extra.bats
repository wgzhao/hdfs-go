#!/usr/bin/env bats

load helper

TEST_DIR=/tmp/hdfs_test_cmd/extra

setup() {
  $HDFS rm -r $TEST_DIR >/dev/null 2>&1 || true
  $HDFS mkdir -p $TEST_DIR/dir/sub
  $HDFS put $ROOT_TEST_DIR/testdata/foo.txt $TEST_DIR/foo.txt
  $HDFS put $ROOT_TEST_DIR/testdata/foo.txt $TEST_DIR/dir/sub/child.txt
  $HDFS put $ROOT_TEST_DIR/testdata/mobydick.txt $TEST_DIR/mobydick.txt
}

@test "appendToFile" {
  printf 'one\n' > "$BATS_TMPDIR/append1"
  printf 'two\n' > "$BATS_TMPDIR/append2"

  run $HDFS appendToFile "$BATS_TMPDIR/append1" $TEST_DIR/appended
  assert_success
  assert_output ""

  run $HDFS appendToFile "$BATS_TMPDIR/append2" $TEST_DIR/appended
  assert_success

  run $HDFS cat $TEST_DIR/appended
  assert_success
  assert_output <<OUT
one
two
OUT
}

@test "appendToFile stdin" {
  run bash -c "printf 'from stdin\n' | $HDFS appendToFile - $TEST_DIR/stdin-appended"
  assert_success

  run $HDFS cat $TEST_DIR/stdin-appended
  assert_output "from stdin"
}

@test "chgrp" {
  group=$($HDFS stat '%g' $TEST_DIR/foo.txt)

  run $HDFS chgrp "$group" $TEST_DIR/foo.txt
  assert_success
  assert_output ""

  run $HDFS stat '%g' $TEST_DIR/foo.txt
  assert_output "$group"
}

@test "count matches hadoop fs" {
  run bash -c "$HDFS count $TEST_DIR > $BATS_TMPDIR/count.ours"
  assert_success
  run bash -c "$HADOOP_FS -count $TEST_DIR > $BATS_TMPDIR/count.theirs"
  assert_success

  run diff -u "$BATS_TMPDIR/count.theirs" "$BATS_TMPDIR/count.ours"
  assert_success
}

@test "count -h and -v match hadoop fs" {
  run bash -c "$HDFS count -h -v $TEST_DIR > $BATS_TMPDIR/count.ours"
  assert_success
  run bash -c "$HADOOP_FS -count -h -v $TEST_DIR > $BATS_TMPDIR/count.theirs"
  assert_success

  run diff -u "$BATS_TMPDIR/count.theirs" "$BATS_TMPDIR/count.ours"
  assert_success
}

@test "cp file" {
  run $HDFS cp $TEST_DIR/foo.txt $TEST_DIR/copied.txt
  assert_success
  assert_output ""

  run $HDFS cat $TEST_DIR/copied.txt
  assert_output "bar"
}

@test "cp file into existing file" {
  run $HDFS cp $TEST_DIR/foo.txt $TEST_DIR/copied.txt
  assert_success

  run $HDFS cp $TEST_DIR/foo.txt $TEST_DIR/copied.txt
  assert_failure

  run $HDFS cp -f $TEST_DIR/foo.txt $TEST_DIR/copied.txt
  assert_success

  run $HDFS cat $TEST_DIR/copied.txt
  assert_output "bar"
}

@test "cp directory" {
  run $HDFS cp $TEST_DIR/dir $TEST_DIR/dircopy
  assert_success

  run $HDFS test -f $TEST_DIR/dircopy/sub/child.txt
  assert_success

  run $HDFS cat $TEST_DIR/dircopy/sub/child.txt
  assert_output "bar"
}

@test "cp -p preserves permissions" {
  $HDFS chmod 640 $TEST_DIR/foo.txt

  run $HDFS cp -p $TEST_DIR/foo.txt $TEST_DIR/preserved.txt
  assert_success

  run $HDFS stat '%a' $TEST_DIR/preserved.txt
  assert_output "640"
}

@test "find matches hadoop fs" {
  run bash -c "$HDFS find $TEST_DIR | sort > $BATS_TMPDIR/find.ours"
  assert_success
  run bash -c "$HADOOP_FS -find $TEST_DIR | sort > $BATS_TMPDIR/find.theirs"
  assert_success

  run diff -u "$BATS_TMPDIR/find.theirs" "$BATS_TMPDIR/find.ours"
  assert_success
}

@test "find -name and -iname" {
  run $HDFS find $TEST_DIR -name '*.txt'
  assert_success
  assert_line "$TEST_DIR/foo.txt"

  run $HDFS find $TEST_DIR -iname 'CHILD.TXT'
  assert_success
  assert_line "$TEST_DIR/dir/sub/child.txt"
}

@test "find -print0" {
  run bash -c "$HDFS find $TEST_DIR -name 'foo.txt' -print0 | tr '\\0' '\\n'"
  assert_success
  assert_output "$TEST_DIR/foo.txt"
}

@test "find unsupported expression" {
  run $HDFS find $TEST_DIR -type f
  assert_failure
  assert_output "find: Unexpected argument: -type"
}

@test "getfattr and setfattr" {
  run $HDFS setfattr -n user.test -v hello $TEST_DIR/foo.txt
  assert_success
  assert_output ""

  run bash -c "$HDFS getfattr -d $TEST_DIR/foo.txt > $BATS_TMPDIR/xattr.ours"
  assert_success
  run bash -c "$HADOOP_FS -getfattr -d $TEST_DIR/foo.txt > $BATS_TMPDIR/xattr.theirs"
  assert_success

  run diff -u "$BATS_TMPDIR/xattr.theirs" "$BATS_TMPDIR/xattr.ours"
  assert_success

  run $HDFS setfattr -x user.test $TEST_DIR/foo.txt
  assert_success
}

@test "getfattr encoding" {
  run $HDFS setfattr -n user.test -v hello $TEST_DIR/foo.txt
  assert_success

  run $HDFS getfattr -d -e hex $TEST_DIR/foo.txt
  assert_success
  assert_line "user.test=0x68656c6c6f"

  run $HDFS getfattr -d -e base64 $TEST_DIR/foo.txt
  assert_success
  assert_line "user.test=0saGVsbG8="

  run $HDFS setfattr -x user.test $TEST_DIR/foo.txt
  assert_success
}

@test "rmdir" {
  run $HDFS mkdir $TEST_DIR/empty
  assert_success

  run $HDFS rmdir $TEST_DIR/empty
  assert_success

  run $HDFS ls $TEST_DIR/empty
  assert_failure
}

@test "rmdir non-empty" {
  run $HDFS mkdir $TEST_DIR/notempty
  run $HDFS touch $TEST_DIR/notempty/file

  run $HDFS rmdir $TEST_DIR/notempty
  assert_failure

  run $HDFS rmdir --ignore-fail-on-non-empty $TEST_DIR/notempty
  assert_success
}

@test "stat matches hadoop fs" {
  format='%a|%A|%b|%F|%g|%n|%o|%r|%u|%y|%Y'

  run bash -c "$HDFS stat '$format' $TEST_DIR/foo.txt > $BATS_TMPDIR/stat.ours"
  assert_success
  run bash -c "$HADOOP_FS -stat '$format' $TEST_DIR/foo.txt > $BATS_TMPDIR/stat.theirs"
  assert_success

  run diff -u "$BATS_TMPDIR/stat.theirs" "$BATS_TMPDIR/stat.ours"
  assert_success
}

@test "stat default format" {
  run bash -c "$HDFS stat $TEST_DIR/foo.txt > $BATS_TMPDIR/stat.ours"
  assert_success
  run bash -c "$HADOOP_FS -stat $TEST_DIR/foo.txt > $BATS_TMPDIR/stat.theirs"
  assert_success

  run diff -u "$BATS_TMPDIR/stat.theirs" "$BATS_TMPDIR/stat.ours"
  assert_success
}

@test "text" {
  run bash -c "$HDFS text $TEST_DIR/foo.txt > $BATS_TMPDIR/text.ours"
  assert_success
  run bash -c "$HADOOP_FS -text $TEST_DIR/foo.txt > $BATS_TMPDIR/text.theirs"
  assert_success

  run diff -u "$BATS_TMPDIR/text.theirs" "$BATS_TMPDIR/text.ours"
  assert_success
}

@test "text gzip" {
  gzip -c $ROOT_TEST_DIR/testdata/foo.txt > "$BATS_TMPDIR/foo.txt.gz"
  $HDFS put "$BATS_TMPDIR/foo.txt.gz" $TEST_DIR/foo.txt.gz

  run bash -c "$HDFS text $TEST_DIR/foo.txt.gz > $BATS_TMPDIR/text.ours"
  assert_success
  run bash -c "$HADOOP_FS -text $TEST_DIR/foo.txt.gz > $BATS_TMPDIR/text.theirs"
  assert_success

  run diff -u "$BATS_TMPDIR/text.theirs" "$BATS_TMPDIR/text.ours"
  assert_success
}

@test "touchz" {
  run $HDFS touchz $TEST_DIR/zero
  assert_success

  run $HDFS stat '%b' $TEST_DIR/zero
  assert_output "0"

  run $HDFS touchz $TEST_DIR/foo.txt
  assert_failure
}

@test "moveFromLocal" {
  cp $ROOT_TEST_DIR/testdata/foo.txt "$BATS_TMPDIR/move-me"

  run $HDFS moveFromLocal "$BATS_TMPDIR/move-me" $TEST_DIR/moved
  assert_success

  run test -e "$BATS_TMPDIR/move-me"
  assert_failure

  run $HDFS cat $TEST_DIR/moved
  assert_output "bar"
}

@test "moveToLocal is not implemented" {
  run $HDFS moveToLocal $TEST_DIR/foo.txt "$BATS_TMPDIR/move-to-local"
  assert_failure
  assert_output "moveToLocal: Option '-moveToLocal' is not implemented yet."
}

@test "copyFromLocal and copyToLocal" {
  cp $ROOT_TEST_DIR/testdata/foo.txt "$BATS_TMPDIR/copy-from-local"

  run $HDFS copyFromLocal "$BATS_TMPDIR/copy-from-local" $TEST_DIR/copied-local
  assert_success

  run $HDFS copyToLocal $TEST_DIR/copied-local "$BATS_TMPDIR/copied-back"
  assert_success

  run cmp "$BATS_TMPDIR/copy-from-local" "$BATS_TMPDIR/copied-back"
  assert_success
}

@test "setrep" {
  run $HDFS setrep 1 $TEST_DIR/foo.txt
  assert_success
  assert_output "Replication 1 set: $TEST_DIR/foo.txt"

  run $HDFS stat '%r' $TEST_DIR/foo.txt
  assert_output "1"
}

@test "usage" {
  run $HDFS usage cp
  assert_success
  assert_output "cp [-fpd] SOURCE... DEST"
}

teardown() {
  $HDFS rm -r $TEST_DIR
}
