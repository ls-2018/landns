import axios from 'axios'


export abstract class Record {
    constructor(
        public readonly name: string,
        public readonly ttl: number = 3600,
        public readonly type: string = 'UNKNOWN',
    ){}

    static parse(str: string): Record {
        const m = str.match(/[^ \t]+[ \t]+[0-9]+[ \t]+IN[ \t]+([A-Z]+)[ \t]/);
        if (m === null) {
            throw new Error(`invalid record: ${str}`);
        }

        const func = {
            A: ARecord.parse,
            AAAA: AaaaRecord.parse,
            CNAME: CnameRecord.parse,
            PTR: PtrRecord.parse,
            TXT: TxtRecord.parse,
            SRV: SrvRecord.parse,
        }[m[1]];

        if (func === undefined) {
            throw new Error(`invalid record: ${str}`);
        }

        return func(str);
    }

    abstract toString(): string;
}


export class ARecord extends Record {
    constructor(
        name: string,
        public readonly address: string,
        ttl: number = 3600,
    ){
        super(name, ttl, 'A');
    }

    static parse(str: string): ARecord {
        const m = str.match(/([^ \t]+)[ \t]+([0-9]+)[ \t]+IN[ \t]+A[ \t]+([^ \t;]+)/);
        if (m === null) {
            throw new Error(`invalid record: ${str}`);
        }

        return new ARecord(m[1], m[3], parseInt(m[2]));
    }

    toString(): string {
        return `${this.name} ${this.ttl} IN A ${this.address}`;
    }
}


export class AaaaRecord extends Record {
    constructor(
        name: string,
        public readonly address: string,
        ttl: number = 3600,
    ){
        super(name, ttl, 'AAAA');
    }

    static parse(str: string): AaaaRecord {
        const m = str.match(/([^ \t]+)[ \t]+([0-9]+)[ \t]+IN[ \t]+AAAA[ \t]+([^ \t;]+)/);
        if (m === null) {
            throw new Error(`invalid record: ${str}`);
        }

        return new AaaaRecord(m[1], m[3], parseInt(m[2]));
    }

    toString(): string {
        return `${this.name} ${this.ttl} IN AAAA ${this.address}`;
    }
}


export class CnameRecord extends Record {
    constructor(
        name: string,
        public readonly target: string,
        ttl: number = 3600,
    ){
        super(name, ttl, 'CNAME');
    }

    static parse(str: string): CnameRecord {
        const m = str.match(/([^ \t]+)[ \t]+([0-9]+)[ \t]+IN[ \t]+CNAME[ \t]+([^ \t;]+)/);
        if (m === null) {
            throw new Error(`invalid record: ${str}`);
        }

        return new CnameRecord(m[1], m[3], parseInt(m[2]));
    }

    toString(): string {
        return `${this.name} ${this.ttl} IN CNAME ${this.target}`;
    }
}


export class PtrRecord extends Record {
    constructor(
        name: string,
        public readonly domain: string,
        ttl: number = 3600,
    ){
        super(name, ttl, 'PTR');
    }

    static parse(str: string): PtrRecord {
        const m = str.match(/([^ \t]+)[ \t]+([0-9]+)[ \t]+IN[ \t]+PTR[ \t]+([^ \t;]+)/);
        if (m === null) {
            throw new Error(`invalid record: ${str}`);
        }

        return new PtrRecord(m[1], m[3], parseInt(m[2]));
    }

    toString(): string {
        return `${this.name} ${this.ttl} IN PTR ${this.domain}`;
    }
}


export class TxtRecord extends Record {
    constructor(
        name: string,
        public readonly text: string,
        ttl: number = 3600,
    ){
        super(name, ttl, 'TXT');
    }

    static parse(str: string): TxtRecord {
        const m = str.match(/([^ \t]+)[ \t]+([0-9]+)[ \t]+IN[ \t]+TXT[ \t]+"([^"]*)"/);
        if (m === null) {
            throw new Error(`invalid record: ${str}`);
        }

        return new TxtRecord(m[1], m[3], parseInt(m[2]));
    }

    toString(): string {
        return `${this.name} ${this.ttl} IN TXT "${this.text}"`;
    }
}


export class SrvRecord extends Record {
    constructor(
        name: string,
        public readonly target: string,
        public readonly port: number,
        public readonly priority: number = 0,
        public readonly weight: number = 0,
        ttl: number = 3600,
    ){
        super(name, ttl, 'SRV');
    }

    static parse(str: string): SrvRecord {
        const m = str.match(/([^ \t]+)[ \t]+([0-9]+)[ \t]+IN[ \t]+SRV[ \t]+([0-9]+)[ \t]+([0-9]+)[ \t]+([0-9]+)[ \t]+([^ \t;]+)/);
        if (m === null) {
            throw new Error(`invalid record: ${str}`);
        }

        return new SrvRecord(m[1], m[6], parseInt(m[5]),  parseInt(m[3]), parseInt(m[4]), parseInt(m[2]));
    }

    toString(): string {
        return `${this.name} ${this.ttl} IN SRV ${this.priority} ${this.weight} ${this.port} ${this.target}`;
    }
}


export function parseRecords(text: string): Record[] {
    return text.split('\n')
        .map(line => line.trim())
        .filter(line => !line.startsWith(';') && line != '')
        .map(line => Record.parse(line))
        .filter(record => record !== null);
}


export class Landns {
    constructor(
        private endpoint: string = 'http://localhost:9353/api/v1',
    ){}

    async set(records: Record[]) {
        await axios.post(
            this.endpoint,
            records.filter(record => record !== null).map(record => record.toString()).join('\n'),
            {headers: {"Content-Type": "text/plain"}},
        );
    }

    async remove(id: number) {
        await axios.delete(`${this.endpoint}/id/${id}`);
    }

    async get(): Promise<Record[]> {
        const resp = await axios.get(this.endpoint);
        return parseRecords(resp.data);
    }

    async glob(query: string): Promise<Record[]> {
        const resp = await axios.get(`${this.endpoint}/glob/${query}`);
        return parseRecords(resp.data);
    }
}
