export default {
    async fetch(request, env) {
        if (request.headers.get("authorization") !== 'foo') {
            return new Response("Sorry, you have supplied an invalid key.", {
                status: 403,
            });
        }

        const body = await request.json();

        const formData = await request.formData();
        const file = formData.get('image');
        const data = await file.arrayBuffer();
        const image = [...new Uint8Array(data)]

        var response = {}
        var now;
        var description;

        const llavaModel = "@cf/llava-hf/llava-1.5-7b-hf"
        const detrModel = "@cf/facebook/detr-resnet-50";
        const translateModel = "@cf/meta/m2m100-1.2b";

        if (request.headers.has('x-llava')) {
            now = new Date();
            response[llavaModel] = await env.AI.run(llavaModel, {
                image: image,
                prompt: "Generate a detailed caption for this image",
                max_tokens: 512,
            });
            description = response[llavaModel].description;
            response[llavaModel].elapsedTimeMs = new Date() - now;
        }


        if (request.headers.has('x-detr')) {
            now = new Date();
            response[detrModel] = await env.AI.run(detrModel, {image: image});
            response[detrModel].elapsedTimeMs = new Date() - now;
        }

        if (request.headers.has("x-translate") && description) {
            const languages = request.headers.get("x-translate").split(",")
            for (var l in languages) {
                now = new Date();
                var lang = languages[l];
                response[translateModel][lang] = await env.AI.run(
                    translateModel,
                    {
                        text: description,
                        source_lang: "english",
                        target_lang: lang,
                    }
                );
                response[translateModel][lang].elapsedTimeMs = new Date() - now;
            }
        }

        return Response.json(response);
    }
};

// time curl -X POST -F 'image=@1zug5643r14w4.jpg' -H "Authorization: foo" -H "x-llava: 1" -H "x-translate: russian,german" -H "x-detr: 1" https://worker-snowy-mode-9777.nanopeni.workers.dev/