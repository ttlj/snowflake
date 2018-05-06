Sonyflake
=========

Sonyflake is a distributed unique ID generator inspired by [Twitter's Snowflake](https://blog.twitter.com/2010/announcing-snowflake).  
A Sonyflake ID is composed of

    - bits for time in units of 1 msec (default: 39)
    - bits for a machine id (default: 16)
    - bits for a sequence number (default: 8)

Characteristics:

    - low-latency uncoordinated, 
    - (roughly) time ordered, 
    - compact and 
    - highly available

Note: This package provides a basis for Id generation; for a RESTful service using this package see TODO.


## System Clock Dependency

It is strongly *recommended* to use NTP to keep your system clock accurate. SnowFlake protects from non-monotonic clocks: it refuses to generate ID if a backwards running clock is detected. However, that protection doesn't work between system/workers restarts.



