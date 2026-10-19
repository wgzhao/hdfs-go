#!/usr/bin/env bats

load helper

@test "version" {
  run $HDFS --version
  assert_success
  assert_match "^gohdfs version " "$output"
}

@test "version shorthand" {
  run $HDFS -v
  assert_success
  assert_match "^gohdfs version " "$output"
}
