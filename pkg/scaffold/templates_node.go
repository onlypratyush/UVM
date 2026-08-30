package scaffold

import "fmt"

// GetNodeTemplates returns the map of relative file path -> file content for Node.js projects
func GetNodeTemplates(projectName, framework, dialect string, includeCRUD bool) map[string]string {
	files := make(map[string]string)
	isTS := dialect == "ts" || dialect == "typescript"

	// 1. Common Files (.gitignore, .env.example)
	files[".gitignore"] = `node_modules/
dist/
build/
.env
.env.local
*.log
npm-debug.log*
yarn-debug.log*
yarn-error.log*
.DS_Store
`
	files[".env.example"] = "PORT=3000\nNODE_ENV=development\n"
	files[".env"] = "PORT=3000\nNODE_ENV=development\n"

	if framework == "fastify" {
		getNodeFastifyTemplates(files, projectName, isTS, includeCRUD)
	} else {
		// Default: Express
		getNodeExpressTemplates(files, projectName, isTS, includeCRUD)
	}

	return files
}

func getNodeExpressTemplates(files map[string]string, projectName string, isTS, includeCRUD bool) {
	if isTS {
		// package.json
		files["package.json"] = fmt.Sprintf(`{
  "name": "%s",
  "version": "1.0.0",
  "description": "Express Clean Architecture API scaffolded by UVM",
  "main": "dist/app.js",
  "scripts": {
    "dev": "ts-node-dev --respawn --transpile-only src/app.ts",
    "build": "tsc",
    "start": "node dist/app.js"
  },
  "dependencies": {
    "cors": "^2.8.5",
    "dotenv": "^16.4.5",
    "express": "^4.19.2"
  },
  "devDependencies": {
    "@types/cors": "^2.8.17",
    "@types/express": "^4.17.21",
    "@types/node": "^20.11.24",
    "ts-node-dev": "^2.0.0",
    "typescript": "^5.3.3"
  }
}
`, projectName)

		// tsconfig.json
		files["tsconfig.json"] = `{
  "compilerOptions": {
    "target": "ES2022",
    "module": "commonjs",
    "lib": ["ES2022"],
    "outDir": "./dist",
    "rootDir": "./src",
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "forceConsistentCasingInFileNames": true,
    "moduleResolution": "node"
  },
  "include": ["src/**/*"]
}
`

		// Types
		files["src/types/item.ts"] = `export interface Item {
  id: string;
  name: string;
  description: string;
  createdAt: Date;
}

export interface CreateItemDTO {
  name: string;
  description: string;
}

export interface UpdateItemDTO {
  name?: string;
  description?: string;
}
`

		// Middleware
		files["src/middleware/errorHandler.ts"] = `import { Request, Response, NextFunction } from 'express';

export function errorHandler(err: Error, req: Request, res: Response, next: NextFunction) {
  console.error('[Error]', err.stack || err.message);
  res.status(500).json({
    success: false,
    error: err.message || 'Internal Server Error',
  });
}
`
		files["src/middleware/logger.ts"] = `import { Request, Response, NextFunction } from 'express';

export function requestLogger(req: Request, res: Response, next: NextFunction) {
  const start = Date.now();
  res.on('finish', () => {
    const duration = Date.now() - start;
    console.log(` + "`" + `[${req.method}] ${req.originalUrl} ${res.statusCode} - ${duration}ms` + "`" + `);
  });
  next();
}
`

		// Service
		files["src/services/itemService.ts"] = `import { Item, CreateItemDTO, UpdateItemDTO } from '../types/item';

const items: Map<string, Item> = new Map();

export const itemService = {
  getAll(): Item[] {
    return Array.from(items.values());
  },

  getById(id: string): Item | undefined {
    return items.get(id);
  },

  create(dto: CreateItemDTO): Item {
    const item: Item = {
      id: Date.now().toString(),
      name: dto.name,
      description: dto.description,
      createdAt: new Date(),
    };
    items.set(item.id, item);
    return item;
  },

  update(id: string, dto: UpdateItemDTO): Item | undefined {
    const item = items.get(id);
    if (!item) return undefined;
    if (dto.name !== undefined) item.name = dto.name;
    if (dto.description !== undefined) item.description = dto.description;
    return item;
  },

  delete(id: string): boolean {
    return items.delete(id);
  },
};
`

		// Controller
		files["src/controllers/itemController.ts"] = `import { Request, Response } from 'express';
import { itemService } from '../services/itemService';

export const itemController = {
  getAll(req: Request, res: Response) {
    const items = itemService.getAll();
    res.json({ success: true, data: items });
  },

  getById(req: Request, res: Response) {
    const item = itemService.getById(req.params.id);
    if (!item) {
      return res.status(404).json({ success: false, error: 'Item not found' });
    }
    res.json({ success: true, data: item });
  },

  create(req: Request, res: Response) {
    const { name, description } = req.body;
    if (!name) {
      return res.status(400).json({ success: false, error: 'Name is required' });
    }
    const newItem = itemService.create({ name, description: description || '' });
    res.status(201).json({ success: true, data: newItem });
  },

  update(req: Request, res: Response) {
    const updated = itemService.update(req.params.id, req.body);
    if (!updated) {
      return res.status(404).json({ success: false, error: 'Item not found' });
    }
    res.json({ success: true, data: updated });
  },

  delete(req: Request, res: Response) {
    const deleted = itemService.delete(req.params.id);
    if (!deleted) {
      return res.status(404).json({ success: false, error: 'Item not found' });
    }
    res.json({ success: true, message: 'Item deleted successfully' });
  },
};
`

		// Routes
		files["src/routes/itemRoutes.ts"] = `import { Router } from 'express';
import { itemController } from '../controllers/itemController';

const router = Router();

router.get('/', itemController.getAll);
router.get('/:id', itemController.getById);
router.post('/', itemController.create);
router.put('/:id', itemController.update);
router.delete('/:id', itemController.delete);

export default router;
`

		// App
		files["src/app.ts"] = `import express from 'express';
import cors from 'cors';
import dotenv from 'dotenv';
import itemRoutes from './routes/itemRoutes';
import { requestLogger } from './middleware/logger';
import { errorHandler } from './middleware/errorHandler';

dotenv.config();

const app = express();
const port = process.env.PORT || 3000;

app.use(cors());
app.use(express.json());
app.use(requestLogger);

app.get('/health', (req, res) => {
  res.json({ status: 'ok', timestamp: new Date().toISOString() });
});

app.use('/api/items', itemRoutes);

app.use(errorHandler);

app.listen(port, () => {
  console.log(` + "`" + `🚀 Server ready at http://localhost:${port}` + "`" + `);
  console.log(` + "`" + `👉 Health Check: http://localhost:${port}/health` + "`" + `);
  console.log(` + "`" + `👉 CRUD API:    http://localhost:${port}/api/items` + "`" + `);
});

export default app;
`

		// README
		files["README.md"] = fmt.Sprintf(`# %s (Express + TypeScript)

Production-grade Express Clean Architecture backend scaffolded by **UVM**.

## 🚀 Quick Start

1. **Install Dependencies**:
   `+"```bash\n   npm install\n   ```"+`
2. **Start Dev Server**:
   `+"```bash\n   npm run dev\n   ```"+`
3. **Build for Production**:
   `+"```bash\n   npm run build\n   npm start\n   ```"+`

## 📡 API Endpoints

| Method | Endpoint | Description |
| :--- | :--- | :--- |
| **GET** | ` + "`/health`" + ` | Service health check |
| **GET** | ` + "`/api/items`" + ` | List all items |
| **GET** | ` + "`/api/items/:id`" + ` | Get single item by ID |
| **POST** | ` + "`/api/items`" + ` | Create new item |
| **PUT** | ` + "`/api/items/:id`" + ` | Update existing item |
| **DELETE** | ` + "`/api/items/:id`" + ` | Delete item |
`, projectName)

	} else {
		// Express JavaScript
		files["package.json"] = fmt.Sprintf(`{
  "name": "%s",
  "version": "1.0.0",
  "description": "Express Clean Architecture API scaffolded by UVM",
  "main": "src/app.js",
  "scripts": {
    "dev": "node --watch src/app.js",
    "start": "node src/app.js"
  },
  "dependencies": {
    "cors": "^2.8.5",
    "dotenv": "^16.4.5",
    "express": "^4.19.2"
  }
}
`, projectName)

		files["src/middleware/errorHandler.js"] = `function errorHandler(err, req, res, next) {
  console.error('[Error]', err.stack || err.message);
  res.status(500).json({
    success: false,
    error: err.message || 'Internal Server Error',
  });
}

module.exports = { errorHandler };
`

		files["src/middleware/logger.js"] = `function requestLogger(req, res, next) {
  const start = Date.now();
  res.on('finish', () => {
    const duration = Date.now() - start;
    console.log(` + "`" + `[${req.method}] ${req.originalUrl} ${res.statusCode} - ${duration}ms` + "`" + `);
  });
  next();
}

module.exports = { requestLogger };
`

		files["src/services/itemService.js"] = `const items = new Map();

const itemService = {
  getAll() {
    return Array.from(items.values());
  },

  getById(id) {
    return items.get(id);
  },

  create(dto) {
    const item = {
      id: Date.now().toString(),
      name: dto.name,
      description: dto.description || '',
      createdAt: new Date(),
    };
    items.set(item.id, item);
    return item;
  },

  update(id, dto) {
    const item = items.get(id);
    if (!item) return null;
    if (dto.name !== undefined) item.name = dto.name;
    if (dto.description !== undefined) item.description = dto.description;
    return item;
  },

  delete(id) {
    return items.delete(id);
  },
};

module.exports = { itemService };
`

		files["src/controllers/itemController.js"] = `const { itemService } = require('../services/itemService');

const itemController = {
  getAll(req, res) {
    const items = itemService.getAll();
    res.json({ success: true, data: items });
  },

  getById(req, res) {
    const item = itemService.getById(req.params.id);
    if (!item) {
      return res.status(404).json({ success: false, error: 'Item not found' });
    }
    res.json({ success: true, data: item });
  },

  create(req, res) {
    const { name, description } = req.body;
    if (!name) {
      return res.status(400).json({ success: false, error: 'Name is required' });
    }
    const newItem = itemService.create({ name, description });
    res.status(201).json({ success: true, data: newItem });
  },

  update(req, res) {
    const updated = itemService.update(req.params.id, req.body);
    if (!updated) {
      return res.status(404).json({ success: false, error: 'Item not found' });
    }
    res.json({ success: true, data: updated });
  },

  delete(req, res) {
    const deleted = itemService.delete(req.params.id);
    if (!deleted) {
      return res.status(404).json({ success: false, error: 'Item not found' });
    }
    res.json({ success: true, message: 'Item deleted successfully' });
  },
};

module.exports = { itemController };
`

		files["src/routes/itemRoutes.js"] = `const { Router } = require('express');
const { itemController } = require('../controllers/itemController');

const router = Router();

router.get('/', itemController.getAll);
router.get('/:id', itemController.getById);
router.post('/', itemController.create);
router.put('/:id', itemController.update);
router.delete('/:id', itemController.delete);

module.exports = router;
`

		files["src/app.js"] = `const express = require('express');
const cors = require('cors');
const dotenv = require('dotenv');
const itemRoutes = require('./routes/itemRoutes');
const { requestLogger } = require('./middleware/logger');
const { errorHandler } = require('./middleware/errorHandler');

dotenv.config();

const app = express();
const port = process.env.PORT || 3000;

app.use(cors());
app.use(express.json());
app.use(requestLogger);

app.get('/health', (req, res) => {
  res.json({ status: 'ok', timestamp: new Date().toISOString() });
});

app.use('/api/items', itemRoutes);

app.use(errorHandler);

app.listen(port, () => {
  console.log(` + "`" + `🚀 Server ready at http://localhost:${port}` + "`" + `);
  console.log(` + "`" + `👉 Health Check: http://localhost:${port}/health` + "`" + `);
  console.log(` + "`" + `👉 CRUD API:    http://localhost:${port}/api/items` + "`" + `);
});

module.exports = app;
`

		files["README.md"] = fmt.Sprintf(`# %s (Express + JavaScript)

Express Clean Architecture API scaffolded by **UVM**.

## 🚀 Quick Start

1. **Install Dependencies**:
   `+"```bash\n   npm install\n   ```"+`
2. **Start Dev Server**:
   `+"```bash\n   npm run dev\n   ```"+`
3. **Start in Production**:
   `+"```bash\n   npm start\n   ```"+`
`, projectName)
	}
}

func getNodeFastifyTemplates(files map[string]string, projectName string, isTS, includeCRUD bool) {
	if isTS {
		files["package.json"] = fmt.Sprintf(`{
  "name": "%s",
  "version": "1.0.0",
  "description": "Fastify High-Performance API scaffolded by UVM",
  "main": "dist/server.js",
  "scripts": {
    "dev": "ts-node-dev --respawn --transpile-only src/server.ts",
    "build": "tsc",
    "start": "node dist/server.js"
  },
  "dependencies": {
    "@fastify/cors": "^9.0.1",
    "@fastify/sensible": "^5.5.0",
    "dotenv": "^16.4.5",
    "fastify": "^4.26.2"
  },
  "devDependencies": {
    "@types/node": "^20.11.24",
    "ts-node-dev": "^2.0.0",
    "typescript": "^5.3.3"
  }
}
`, projectName)

		files["tsconfig.json"] = `{
  "compilerOptions": {
    "target": "ES2022",
    "module": "commonjs",
    "lib": ["ES2022"],
    "outDir": "./dist",
    "rootDir": "./src",
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "forceConsistentCasingInFileNames": true,
    "moduleResolution": "node"
  },
  "include": ["src/**/*"]
}
`

		files["src/schemas/itemSchema.ts"] = `export const itemSchema = {
  type: 'object',
  properties: {
    id: { type: 'string' },
    name: { type: 'string' },
    description: { type: 'string' },
    createdAt: { type: 'string' },
  },
};

export const createItemSchema = {
  body: {
    type: 'object',
    required: ['name'],
    properties: {
      name: { type: 'string' },
      description: { type: 'string' },
    },
  },
};
`

		files["src/routes/itemRoutes.ts"] = `import { FastifyPluginAsync } from 'fastify';
import { itemSchema, createItemSchema } from '../schemas/itemSchema';

interface Item {
  id: string;
  name: string;
  description: string;
  createdAt: string;
}

const items: Map<string, Item> = new Map();

const itemRoutes: FastifyPluginAsync = async (fastify) => {
  fastify.get('/', async () => {
    return Array.from(items.values());
  });

  fastify.get<{ Params: { id: string } }>('/:id', async (req, reply) => {
    const item = items.get(req.params.id);
    if (!item) return reply.notFound('Item not found');
    return item;
  });

  fastify.post<{ Body: { name: string; description?: string } }>('/', { schema: createItemSchema }, async (req, reply) => {
    const { name, description } = req.body;
    const newItem: Item = {
      id: Date.now().toString(),
      name,
      description: description || '',
      createdAt: new Date().toISOString(),
    };
    items.set(newItem.id, newItem);
    reply.status(201);
    return newItem;
  });

  fastify.delete<{ Params: { id: string } }>('/:id', async (req, reply) => {
    const deleted = items.delete(req.params.id);
    if (!deleted) return reply.notFound('Item not found');
    return { success: true, message: 'Item deleted' };
  });
};

export default itemRoutes;
`

		files["src/server.ts"] = `import Fastify from 'fastify';
import cors from '@fastify/cors';
import sensible from '@fastify/sensible';
import dotenv from 'dotenv';
import itemRoutes from './routes/itemRoutes';

dotenv.config();

const fastify = Fastify({ logger: true });
const port = Number(process.env.PORT) || 3000;

async function bootstrap() {
  await fastify.register(cors);
  await fastify.register(sensible);

  fastify.get('/health', async () => ({
    status: 'ok',
    timestamp: new Date().toISOString(),
  }));

  await fastify.register(itemRoutes, { prefix: '/api/items' });

  try {
    await fastify.listen({ port, host: '0.0.0.0' });
    console.log(` + "`" + `🚀 Fastify ready at http://localhost:${port}` + "`" + `);
  } catch (err) {
    fastify.log.error(err);
    process.exit(1);
  }
}

bootstrap();
`

		files["README.md"] = fmt.Sprintf(`# %s (Fastify + TypeScript)

High-performance Fastify REST API scaffolded by **UVM**.

## 🚀 Quick Start

1. **Install Dependencies**:
   `+"```bash\n   npm install\n   ```"+`
2. **Start Dev Server**:
   `+"```bash\n   npm run dev\n   ```"+`
3. **Build & Run**:
   `+"```bash\n   npm run build\n   npm start\n   ```"+`
`, projectName)

	} else {
		// Fastify JavaScript
		files["package.json"] = fmt.Sprintf(`{
  "name": "%s",
  "version": "1.0.0",
  "description": "Fastify High-Performance API scaffolded by UVM",
  "main": "src/server.js",
  "scripts": {
    "dev": "node --watch src/server.js",
    "start": "node src/server.js"
  },
  "dependencies": {
    "@fastify/cors": "^9.0.1",
    "@fastify/sensible": "^5.5.0",
    "dotenv": "^16.4.5",
    "fastify": "^4.26.2"
  }
}
`, projectName)

		files["src/routes/itemRoutes.js"] = `const items = new Map();

async function itemRoutes(fastify) {
  fastify.get('/', async () => {
    return Array.from(items.values());
  });

  fastify.get('/:id', async (req, reply) => {
    const item = items.get(req.params.id);
    if (!item) return reply.notFound('Item not found');
    return item;
  });

  fastify.post('/', async (req, reply) => {
    const { name, description } = req.body || {};
    if (!name) return reply.badRequest('Name is required');
    const newItem = {
      id: Date.now().toString(),
      name,
      description: description || '',
      createdAt: new Date().toISOString(),
    };
    items.set(newItem.id, newItem);
    reply.status(201);
    return newItem;
  });

  fastify.delete('/:id', async (req, reply) => {
    const deleted = items.delete(req.params.id);
    if (!deleted) return reply.notFound('Item not found');
    return { success: true, message: 'Item deleted' };
  });
}

module.exports = itemRoutes;
`

		files["src/server.js"] = `const Fastify = require('fastify');
const cors = require('@fastify/cors');
const sensible = require('@fastify/sensible');
const dotenv = require('dotenv');
const itemRoutes = require('./routes/itemRoutes');

dotenv.config();

const fastify = Fastify({ logger: true });
const port = Number(process.env.PORT) || 3000;

async function bootstrap() {
  await fastify.register(cors);
  await fastify.register(sensible);

  fastify.get('/health', async () => ({
    status: 'ok',
    timestamp: new Date().toISOString(),
  }));

  await fastify.register(itemRoutes, { prefix: '/api/items' });

  try {
    await fastify.listen({ port, host: '0.0.0.0' });
    console.log(` + "`" + `🚀 Fastify ready at http://localhost:${port}` + "`" + `);
  } catch (err) {
    fastify.log.error(err);
    process.exit(1);
  }
}

bootstrap();
`

		files["README.md"] = fmt.Sprintf(`# %s (Fastify + JavaScript)

Fastify High-Performance API scaffolded by **UVM**.

## 🚀 Quick Start

1. **Install Dependencies**:
   `+"```bash\n   npm install\n   ```"+`
2. **Start Dev Server**:
   `+"```bash\n   npm run dev\n   ```"+`
3. **Start in Production**:
   `+"```bash\n   npm start\n   ```"+`
`, projectName)
	}
}
