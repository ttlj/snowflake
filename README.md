SnowFlake
=========

SnowFlake is a distributed unique ID generator inspired by [Twitter's Snowflake](https://blog.twitter.com/2010/announcing-snowflake).

## Packages

- snowflake: ID generator
- restful: exposes the ID generator as RESTful service
- kflake: main package using `restful`

## Composition

An ID is composed of:

    - bits for time in units of 1 msec (default: 39)
    - bits for a machine id (default: 16)
    - bits for a sequence number (default: 8)

However, using the `MaskConfig` structure you can tune the structure of the created Id's to your own needs. For example, using 42 bits for the timestamp (≈139 years), 5 bits for the generator-id and 16 bits for the sequence will generate 65536 id's per millisecond per generator; or over 2 million is'd per millisecond when distributed over 32 workers.

Note: The default `MaskConfig` was choosen 

### Characteristics

    - low-latency uncoordinated, 
    - (roughly) time ordered, 
    - compact and 
    - highly available

Note: This package provides a basis for Id generation; for a RESTful service using this package see TODO.


## System Clock Dependency

It is strongly *recommended* to use NTP to keep your system clock accurate. SnowFlake protects from non-monotonic clocks: it refuses to generate ID if a backwards running clock is detected. However, that protection doesn't work between system/workers restarts.



## kflake

Example package that can be used to deploy in a Kubernetes environment.

TODO: yaml files and completed docs
