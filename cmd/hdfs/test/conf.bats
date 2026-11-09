#!/usr/bin/env bats

load helper

# These tests exercise the configuration-directory search order without
# needing a reachable cluster: the fake configuration points at port 1, so
# successfully finding it produces a connection error, while failing to find
# it produces "Couldn't find a namenode to connect to".

setup() {
  HADOOP_TEST_HOME="$(mktemp -d)"
}

teardown() {
  rm -rf "$HADOOP_TEST_HOME"
}

write_conf() {
  mkdir -p "$1"
  cat > "$1/core-site.xml" <<EOF
<?xml version="1.0"?>
<configuration>
  <property>
    <name>fs.defaultFS</name>
    <value>hdfs://127.0.0.1:1</value>
  </property>
</configuration>
EOF
}

assert_conf_found() {
  case "$output" in
    *"Couldn't find a namenode"*)
      flunk "configuration was not found. output: $output"
      ;;
  esac
}

@test "configuration is loaded from HADOOP_HOME/etc/hadoop" {
  write_conf "$HADOOP_TEST_HOME/etc/hadoop"

  run env -u HADOOP_CONF_DIR -u HADOOP_NAMENODE HADOOP_HOME="$HADOOP_TEST_HOME" $HDFS ls /
  assert_failure
  assert_match "Couldn't connect to namenode" "$output"
  assert_conf_found
}

@test "configuration is loaded from HADOOP_HOME/conf" {
  write_conf "$HADOOP_TEST_HOME/conf"

  run env -u HADOOP_CONF_DIR -u HADOOP_NAMENODE HADOOP_HOME="$HADOOP_TEST_HOME" $HDFS ls /
  assert_failure
  assert_match "Couldn't connect to namenode" "$output"
  assert_conf_found
}

@test "an empty HADOOP_CONF_DIR falls through to HADOOP_HOME/etc/hadoop" {
  mkdir -p "$HADOOP_TEST_HOME/empty"
  write_conf "$HADOOP_TEST_HOME/etc/hadoop"

  run env -u HADOOP_NAMENODE \
    HADOOP_CONF_DIR="$HADOOP_TEST_HOME/empty" \
    HADOOP_HOME="$HADOOP_TEST_HOME" \
    $HDFS ls /
  assert_failure
  assert_match "Couldn't connect to namenode" "$output"
  assert_conf_found
}

@test "configuration is loaded from /etc/hadoop/conf when the environment is unset" {
  if [ ! -f /etc/hadoop/conf/core-site.xml ]; then
    skip "no /etc/hadoop/conf/core-site.xml on this system"
  fi

  run env -u HADOOP_CONF_DIR -u HADOOP_HOME -u HADOOP_NAMENODE $HDFS ls /
  assert_conf_found
}
