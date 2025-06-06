To use `asnames`, you'll need to install the dependencies. You can do that with the following:

```shell
python3 -m venv venv
source venv/bin/activate
pip install -r requirements.txt
```

Then you can run with:

```shell
python3 ./
```

The output is a JSON-lines file, so can be saved with:

```shell
python3 ./ > asnames.jsonl
```