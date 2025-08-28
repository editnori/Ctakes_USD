package org.hsqldb;

// Compatibility shim: cTAKES DictionaryBuilder expects the legacy 1.8 driver class
// name org.hsqldb.jdbcDriver. This class delegates to the modern 2.x driver
// org.hsqldb.jdbc.JDBCDriver that is bundled with cTAKES.
public class jdbcDriver extends org.hsqldb.jdbc.JDBCDriver {}

