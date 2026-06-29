<script>
    import { Collection } from "pocketbase";
    import ApiClient from "@/utils/ApiClient";
    import CommonHelper from "@/utils/CommonHelper";
    import CodeBlock from "@/components/base/CodeBlock.svelte";
    import SdkTabs from "@/components/collections/docs/SdkTabs.svelte";

    export let collection = new Collection();

    let responseTab = 200;
    let responses = [];

    $: backendAbsUrl = CommonHelper.getApiExampleUrl(ApiClient.baseUrl);

    $: if (collection?.id) {
        responses.push({
            code: 200,
            body: JSON.stringify(
                {
                    items: [
                        {
                            ...CommonHelper.dummyCollectionRecord(collection),
                            _distance: 0.0825
                        },
                        {
                            ...CommonHelper.dummyCollectionRecord(collection),
                            _distance: 0.1542
                        }
                    ],
                    totalItems: 2
                },
                null,
                2
            ),
        });

        responses.push({
            code: 400,
            body: `
                {
                  "code": 400,
                  "message": "Failed to parse request body or generate embedding.",
                  "data": {}
                }
            `,
        });
    }
</script>

<h3 class="m-b-sm">Vector Search ({collection.name})</h3>
<div class="content txt-lg m-b-sm">
    <p>
        Perform a high-performance vector similarity search (ANN) using <strong>sqlite-vec</strong>.
        You can search either by providing a plain text <code>query</code> (which is embedded automatically on the server side using the configured embedding model) or by passing a raw float array <code>vector</code>.
    </p>
</div>

<SdkTabs
    js={`
        import PocketBase from 'pocketbase';

        const pb = new PocketBase('${backendAbsUrl}');

        // Option A: Search by text query (backend auto-embeds)
        const resultText = await pb.send('/api/collections/${collection?.name}/vector-search', {
            method: 'POST',
            body: {
                query: 'Search query to compare against the vector field',
                limit: 10,
                distance: 'cosine' // or 'l2'
            }
        });

        // Option B: Search by raw embedding vector
        const resultVector = await pb.send('/api/collections/${collection?.name}/vector-search', {
            method: 'POST',
            body: {
                vector: [0.0123, -0.4567, 0.8901, ...], // Must match collection vector dimension
                limit: 5,
                distance: 'cosine'
            }
        });
    `}
    dart={`
        import 'package:pocketbase/pocketbase.dart';

        final pb = PocketBase('${backendAbsUrl}');

        // Option A: Search by text query
        final resultText = await pb.client.send(
          '/api/collections/${collection?.name}/vector-search',
          method: 'POST',
          body: {
            'query': 'Search query to compare against the vector field',
            'limit': 10,
            'distance': 'cosine',
          },
        );

        // Option B: Search by raw embedding vector
        final resultVector = await pb.client.send(
          '/api/collections/${collection?.name}/vector-search',
          method: 'POST',
          body: {
            'vector': [0.0123, -0.4567, 0.8901],
            'limit': 5,
            'distance': 'cosine',
          },
        );
    `}
/>

<h4 class="m-t-lg m-b-sm">API Specifications</h4>

<div class="api-route alert alert-info">
    <strong class="label label-info">POST</strong>
    <code>/api/collections/<strong>{collection.name}</strong>/vector-search</code>
</div>

<p>You can also use GET with query parameters, although POST is highly recommended for long vectors.</p>

<div class="api-route alert alert-info">
    <strong class="label label-info">GET</strong>
    <code>/api/collections/<strong>{collection.name}</strong>/vector-search</code>
</div>

<h4 class="m-t-md m-b-sm">Request Body (JSON) / Query Parameters</h4>
<table class="table-api">
    <thead>
        <tr>
            <th style="width: 150px;">Parameter</th>
            <th style="width: 100px;">Type</th>
            <th>Description</th>
        </tr>
    </thead>
    <tbody>
        <tr>
            <td>vector</td>
            <td>
                <span class="label">Array&lt;Number&gt;</span>
            </td>
            <td>
                The raw query embedding float array (e.g. <code>[0.12, 0.34, ...]</code>) to compare.
            </td>
        </tr>
        <tr>
            <td>query</td>
            <td>
                <span class="label">String</span>
            </td>
            <td>
                A plain text query string. If provided, the backend will automatically call the configured global embedding model to generate the vector on-the-fly.
            </td>
        </tr>
        <tr>
            <td>limit</td>
            <td>
                <span class="label">Number</span>
            </td>
            <td>
                The maximum number of nearest records to return (default to <code>10</code>, max <code>100</code>).
            </td>
        </tr>
        <tr>
            <td>distance</td>
            <td>
                <span class="label">String</span>
            </td>
            <td>
                The distance metric algorithm: <code>cosine</code> (Cosine Distance) or <code>l2</code> (L2 / Euclidean Distance). Default is <code>cosine</code>.
            </td>
        </tr>
        <tr>
            <td>field</td>
            <td>
                <span class="label">String</span>
            </td>
            <td>
                The vector field name in the schema. If omitted, the first vector field found in the collection schema is selected.
            </td>
        </tr>
    </tbody>
</table>

<h4 class="m-t-lg m-b-sm">Responses</h4>
<div class="tabs">
    <div class="tabs-header">
        {#each responses as r}
            <button
                type="button"
                class="tab-item"
                class:active={responseTab === r.code}
                on:click={() => (responseTab = r.code)}
            >
                {r.code}
            </button>
        {/each}
    </div>
    <div class="tabs-content">
        {#each responses as r}
            {#if responseTab === r.code}
                <CodeBlock content={r.body} />
            {/if}
        {/each}
    </div>
</div>
