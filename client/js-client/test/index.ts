import assert = require('assert');
import {Record, ARecord, AaaaRecord, CnameRecord, PtrRecord, TxtRecord, SrvRecord, parseRecords} from '../src';

describe('Record', () => {
    it('ARecord', () => {
        const r = ARecord.parse('example.com. 123 IN A 127.0.0.1');
        assert(r !== null);

        assert(r.name === 'example.com.');
        assert(r.address === '127.0.0.1');
        assert(r.ttl === 123);
        assert(r.toString() === 'example.com. 123 IN A 127.0.0.1');

        assert.throws(
            () => ARecord.parse('hello world'),
            e => e.message === 'invalid record: hello world',
        );
    });

    it('AaaaRecord', () => {
        const r = AaaaRecord.parse('example.com. 123 IN AAAA 4::2');
        assert(r !== null);

        assert(r.name === 'example.com.');
        assert(r.address === '4::2');
        assert(r.ttl === 123);
        assert(r.toString() === 'example.com. 123 IN AAAA 4::2');

        assert.throws(
            () => AaaaRecord.parse('hello world'),
            e => e.message === 'invalid record: hello world',
        );
    });

    it('CnameRecord', () => {
        const r = CnameRecord.parse('example.com. 123 IN CNAME test.local.');
        assert(r !== null);

        assert(r.name === 'example.com.');
        assert(r.target === 'test.local.');
        assert(r.ttl === 123);
        assert(r.toString() === 'example.com. 123 IN CNAME test.local.');

        assert.throws(
            () => CnameRecord.parse('hello world'),
            e => e.message === 'invalid record: hello world',
        );
    });

    it('PtrRecord', () => {
        const r = PtrRecord.parse('1.0.0.127.in-addr.arpa. 123 IN PTR example.com.');
        assert(r !== null);

        assert(r.name === '1.0.0.127.in-addr.arpa.');
        assert(r.domain === 'example.com.');
        assert(r.ttl === 123);
        assert(r.toString() === '1.0.0.127.in-addr.arpa. 123 IN PTR example.com.');

        assert.throws(
            () => PtrRecord.parse('hello world'),
            e => e.message === 'invalid record: hello world',
        );
    });

    it('TxtRecord', () => {
        const r = TxtRecord.parse('example.com. 123 IN TXT "hello world"');
        assert(r !== null);

        assert(r.name === 'example.com.');
        assert(r.text === 'hello world');
        assert(r.ttl === 123);
        assert(r.toString() === 'example.com. 123 IN TXT "hello world"');

        assert.throws(
            () => TxtRecord.parse('hello world'),
            e => e.message === 'invalid record: hello world',
        );
    });

    it('SrvRecord', () => {
        const r = SrvRecord.parse('example.com. 123 IN SRV 1 2 3 test.local.');
        assert(r !== null);

        assert(r.name === 'example.com.');
        assert(r.target === 'test.local.');
        assert(r.priority === 1);
        assert(r.weight === 2);
        assert(r.port === 3);
        assert(r.ttl === 123);
        assert(r.toString() === 'example.com. 123 IN SRV 1 2 3 test.local.');

        assert.throws(
            () => SrvRecord.parse('hello world'),
            e => e.message === 'invalid record: hello world',
        );
    });

    it('Record', () => {
        assert.throws(
            () => Record.parse('hello world'),
            e => e.message === 'invalid record: hello world',
        );
        assert.throws(
            () => Record.parse('example.com. 42 IN UNKNOWN foobar'),
            e => e.message === 'invalid record: example.com. 42 IN UNKNOWN foobar',
        );
    });

    it('parseRecords', () => {
        interface Test {
            input: string;
            expect: string[] | null;
        }

        const tests: Test[] = [
            {input: 'a.example.com.          123 IN A \t  127.0.1.2', expect: null},
            {input: 'b.example.com.          456 IN AAAA  1:2:3::4', expect: null},
            {input: 'c.\t\t\t\t\t\t\t\t\t\t\t789 IN CNAME a.example.com.', expect: null},
            {input: '1.0.0.127.in-addr.arpa. 111 IN PTR   d.local.', expect: null},
            {input: 'e.example.com.          222 IN TXT   "hello world!"', expect: null},
            {input: 'f.f.f.f.com.            333 IN SRV   10 20 30  example.com.', expect: null},
            {
                input: [
                    'a.com. 1 IN A 127.1.1.1',
                    '; this is comment',
                    '  ; stil comment',
                    'b.com. 2 IN A 127.2.2.2 ; comment',
                    '',
                    'c.com. 3 IN A 127.3.3.3;comment too',
                    '',
                    '',
                ].join('\n'),
                expect: ['a.com. 1 IN A 127.1.1.1', 'b.com. 2 IN A 127.2.2.2', 'c.com. 3 IN A 127.3.3.3'],
            },
        ]

        tests.forEach(tt => {
            const rs = parseRecords(tt.input);
            assert(rs !== null);

            const got = rs.map(x => x.toString());

            if (tt.expect === null) {
                tt.expect = [tt.input.replace(/[ \t]+/g, ' ')]
            }

            assert.deepEqual(got, tt.expect);
        });
    });
});
