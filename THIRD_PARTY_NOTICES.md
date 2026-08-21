# Third-party notices and provenance

CPAMP Theme Studio is distributed under the MIT License in [LICENSE](LICENSE).

## CLIProxyAPI / CPA

The plugin implements the public CLIProxyAPI plugin ABI and management-resource JSON contract. The C ABI declarations follow the official CLIProxyAPI SDK and examples. CLIProxyAPI is MIT-licensed by Luis Pater and Router-For.ME. No CLIProxyAPI executable or source tree is included in this repository or its plugin archives.

Project: <https://github.com/router-for-me/CLIProxyAPI>

## CPA Manager Plus / CPAMP

This standalone project grew from a Theme Studio prototype authored for CPA Manager Plus. CPA Manager Plus is MIT-licensed by Seakee. The plugin does not redistribute `management.html`; at runtime it adds and removes a small loader block in a user-supplied, writable panel file.

Project: <https://github.com/seakee/CPA-Manager-Plus>

## gopkg.in/yaml.v3

The compiled plugin includes `gopkg.in/yaml.v3` v3.0.1 for safe configuration parsing. That module is dual-covered by MIT and Apache License 2.0 terms; see its upstream license for the per-file split.

Project: <https://gopkg.in/yaml.v3>

### Upstream license and notice (verbatim)

The following text is reproduced from `gopkg.in/yaml.v3` v3.0.1.

> This project is covered by two different licenses: MIT and Apache.
>
> #### MIT License
>
> The following files were ported to Go from C files of libyaml, and thus
> are still covered by their original MIT license, with the additional
> copyright staring in 2011 when the project was ported over:
>
>     apic.go emitterc.go parserc.go readerc.go scannerc.go
>     writerc.go yamlh.go yamlprivateh.go
>
> Copyright (c) 2006-2010 Kirill Simonov
> Copyright (c) 2006-2011 Kirill Simonov
>
> Permission is hereby granted, free of charge, to any person obtaining a copy of
> this software and associated documentation files (the "Software"), to deal in
> the Software without restriction, including without limitation the rights to
> use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies
> of the Software, and to permit persons to whom the Software is furnished to do
> so, subject to the following conditions:
>
> The above copyright notice and this permission notice shall be included in all
> copies or substantial portions of the Software.
>
> THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
> IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
> FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
> AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
> LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
> OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
> SOFTWARE.
>
> #### Apache License
>
> All the remaining project files are covered by the Apache license:
>
> Copyright (c) 2011-2019 Canonical Ltd
>
> Licensed under the Apache License, Version 2.0 (the "License");
> you may not use this file except in compliance with the License.
> You may obtain a copy of the License at
>
>     http://www.apache.org/licenses/LICENSE-2.0
>
> Unless required by applicable law or agreed to in writing, software
> distributed under the License is distributed on an "AS IS" BASIS,
> WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
> See the License for the specific language governing permissions and
> limitations under the License.

Upstream `NOTICE`:

> Copyright 2011-2016 Canonical Ltd.
>
> Licensed under the Apache License, Version 2.0 (the "License");
> you may not use this file except in compliance with the License.
> You may obtain a copy of the License at
>
>     http://www.apache.org/licenses/LICENSE-2.0
>
> Unless required by applicable law or agreed to in writing, software
> distributed under the License is distributed on an "AS IS" BASIS,
> WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
> See the License for the specific language governing permissions and
> limitations under the License.

## Palette provenance

The palettes shipped by this repository were independently defined for CPAMP Theme Studio. The project does not include code, palette values, artwork, or other assets from the AGPL-3.0 `new-api` project. A small compatibility map retains old preset identifiers only so existing local browser preferences can migrate to the new independently named presets.
