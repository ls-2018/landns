Landns Client Library for JS
============================

The client library for [Landns](https://github.com/macrat/landns).


## Usage

``` javascript
import {Landns, parseRecords} from 'landns';


const client = new Landns();

client.set(parseRecords(`
    example.com. 123 IN A 127.0.0.1
    example.com. 123 IN A 127.0.0.2
`));

const records = client.glob("*.example.com");
records.forEach(record => {
    console.log(record.toString());
});
```
