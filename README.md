<p align="center">
  <img src="https://avatars.githubusercontent.com/u/193303345" alt="Juice Logo" width="200" height="auto"/>
</p>

## Juice: A SQL Mapper for Go Inspired by MyBatis

[![Go Doc](https://pkg.go.dev/badge/github.com/go-juicedev/juice)](https://pkg.go.dev/github.com/go-juicedev/juice)
[![Release](https://img.shields.io/github/v/release/eatmoreapple/juice.svg?style=flat-square)](https://github.com/go-juicedev/juice/releases)
![Go Report Card](https://goreportcard.com/badge/github.com/go-juicedev/juice)
![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)
[![JetBrains Marketplace](https://img.shields.io/jetbrains/plugin/v/26401-juice.svg)](https://plugins.jetbrains.com/plugin/26401-juice)
[![JetBrains Marketplace Downloads](https://img.shields.io/jetbrains/plugin/d/26401-juice.svg)](https://plugins.jetbrains.com/plugin/26401-juice)

Juice is a SQL mapper for Go that keeps SQL explicit while adding XML mappers, dynamic SQL, typed binding, middleware, and transaction helpers.

- [Why Juice](#why-juice)
- [Quick Start](#quick-start)
- [Installation](#installation)
- [When To Use](#when-to-use)
- [Documentation](#documentation)
- [License](#license)
- [Support Me](#support-me)

### Why Juice

- Keep SQL readable and explicit instead of hiding it behind a heavy ORM
- Organize queries with XML mappers inspired by MyBatis
- Build dynamic SQL with nodes like `if`, `where`, `set`, and `foreach`
- Bind query results into typed Go values with generics
- Extend execution with middleware, transaction helpers, and datasource switching

### Quick Start

Create a minimal configuration:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE configuration PUBLIC "-//juice.org//DTD Config 1.0//EN"
        "https://raw.githubusercontent.com/go-juicedev/juice/refs/heads/main/config.dtd">

<configuration>
    <environments default="prod">
        <environment id="prod">
            <dataSource>sqlite.db</dataSource>
            <driver>sqlite3</driver>
        </environment>
    </environments>

    <mappers>
        <mapper resource="mappers.xml"/>
    </mappers>
</configuration>
```

Define a mapper:

```xml
<?xml version="1.0" encoding="utf-8" ?>
<!DOCTYPE mapper PUBLIC "-//juice.org//DTD Config 1.0//EN"
        "https://raw.githubusercontent.com/go-juicedev/juice/refs/heads/main/mapper.dtd">

<mapper namespace="main.Repository">
    <select id="HelloWorld">
        <if test="1 == 1">
            select "hello world"
        </if>
    </select>
</mapper>
```

Call it from Go:

```go
package main

import (
	"context"
	"fmt"

	"github.com/go-juicedev/juice"
	_ "github.com/mattn/go-sqlite3"
)

type Repository interface {
	HelloWorld(ctx context.Context) (string, error)
}

type RepositoryImpl struct {
	manager juice.Manager
}

func (r RepositoryImpl) HelloWorld(ctx context.Context) (string, error) {
	executor := juice.NewGenericManager[string](r.manager).Object(Repository(r).HelloWorld)
	return executor.QueryContext(ctx, nil)
}

func main() {
	cfg, err := juice.NewXMLConfiguration("config.xml")
	if err != nil {
		panic(err)
	}

	engine, err := juice.Default(cfg)
	if err != nil {
		panic(err)
	}
	defer engine.Close()

	repo := RepositoryImpl{manager: engine}
	result, err := repo.HelloWorld(context.Background())
	fmt.Println(result, err)
}
```

Run it:

```sh
CGO_ENABLED=1 go run main.go
```

Expected output:

```text
hello world <nil>
```

### Installation

Install Juice:

```sh
go get github.com/go-juicedev/juice
```

Install a database driver for the backend you want to use. For example, the quick start above uses SQLite:

```go
import _ "github.com/mattn/go-sqlite3"
```

If you use the SQLite example above, `CGO_ENABLED=1` is required.

### When To Use

Juice is a good fit when:

- you want explicit SQL instead of a code-first ORM
- you like MyBatis-style mapper organization
- you need dynamic SQL without manually concatenating strings
- you want a thin abstraction over `database/sql` with stronger structure

Juice may not be the right fit when:

- you want a code-first query builder or ORM model layer
- you do not want XML-based mapping at all
- you expect built-in migration or schema management features

### Documentation

- API Reference: <https://pkg.go.dev/github.com/go-juicedev/juice>
- English Docs: <https://juice-doc.readthedocs.io/projects/juice-doc-en/en/latest/>
- 简体中文文档: <https://juice-doc.readthedocs.io/en/latest/index.html>

### License

Juice is licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for the full license text.

## Support Me

If you like my work, please consider supporting me by buying me a coffee.

<a href="https://raw.githubusercontent.com/eatmoreapple/eatmoreapple/main/img/wechat_pay.jpg" target="_blank"><img src="https://cdn.buymeacoffee.com/buttons/v2/default-yellow.png" alt="Buy Me A Coffee" width="150" ></a>
