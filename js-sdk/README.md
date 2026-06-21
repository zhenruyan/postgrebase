# PostgreBase JavaScript SDK

JavaScript SDK (browser and node) for interacting with the [PostgreBase API](https://github.com/zhenruyan/postgrebase).

Based on the [PocketBase JS SDK](https://github.com/pocketbase/js-sdk) by [Gani Georgiev](https://github.com/ganigeorgiev).

## Installation

### Browser (manually via script tag)

```html
<script src="/path/to/dist/postgrebase.umd.js"></script>
<script type="text/javascript">
    const pb = new PostgreBase("https://example.com")
    ...
</script>
```

_OR if you are using ES modules:_
```html
<script type="module">
    import PostgreBase from '/path/to/dist/postgrebase.es.mjs'

    const pb = new PostgreBase("https://example.com")
    ...
</script>
```

### Node.js (via npm)

```sh
npm install postgrebase --save
```

```js
// Using ES modules (default)
import PostgreBase from 'postgrebase'

// OR if you are using CommonJS modules
const PostgreBase = require('postgrebase/cjs')
```

> 🔧 For **Node < 17** you'll need to load a `fetch()` polyfill.
> I recommend [lquixada/cross-fetch](https://github.com/lquixada/cross-fetch):
> ```js
> // npm install cross-fetch --save
> import 'cross-fetch/polyfill';
> ```
---
> 🔧 Node doesn't have native `EventSource` implementation, so in order to use the realtime subscriptions you'll need to load a `EventSource` polyfill.
> ```js
> // for server: npm install eventsource --save
> import eventsource from 'eventsource';
>
> // for React Native: npm install react-native-sse --save
> import eventsource from "react-native-sse";
>
> global.EventSource = eventsource;
> ```

## Usage

```js
import PostgreBase from 'postgrebase';

const pb = new PostgreBase('http://127.0.0.1:8090');

...

// list and filter "example" collection records
const result = await pb.collection('example').getList(1, 20, {
    filter: 'status = true && created > "2022-08-01 10:00:00"'
});

// authenticate as auth collection record
const userData = await pb.collection('users').authWithPassword('test@example.com', '123456');

// or as super-admin
const adminData = await pb.admins.authWithPassword('test@example.com', '123456');

// and much more...
```

## API Documentation

For detailed API docs and examples, see the [PostgreBase README](https://github.com/zhenruyan/postgrebase/blob/main/README.md).

## Features

- 🗄️ **Multi-Database**: Works with PostgreSQL, MySQL, and SQLite backends
- 🔄 **Realtime**: Subscribe to record changes via SSE
- 🔐 **Auth**: Built-in authentication helpers
- 📁 **Files**: File upload and download support
- 🔍 **Filtering**: Powerful filter expressions
- 📄 **Pagination**: Built-in pagination support

## Development

```bash
# Install dependencies
npm install

# Build
npm run build

# Run tests
npm test

# Development mode (watch)
npm run dev
```

## License

MIT — Based on [PocketBase JS SDK](https://github.com/pocketbase/js-sdk) by [Gani Georgiev](https://github.com/ganigeorgiev)
