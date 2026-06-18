export namespace config {
	
	export class Config {
	    theme: string;
	    output_dir: string;
	    build_speed: string;
	    build_scope_mode: string;
	    last_branch: string;
	    smart_dependency: boolean;
	    project_root: string;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.theme = source["theme"];
	        this.output_dir = source["output_dir"];
	        this.build_speed = source["build_speed"];
	        this.build_scope_mode = source["build_scope_mode"];
	        this.last_branch = source["last_branch"];
	        this.smart_dependency = source["smart_dependency"];
	        this.project_root = source["project_root"];
	    }
	}

}

export namespace git {
	
	export class Commit {
	    hash: string;
	    repo?: string;
	    author: string;
	    date: string;
	    msg: string;
	
	    static createFrom(source: any = {}) {
	        return new Commit(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hash = source["hash"];
	        this.repo = source["repo"];
	        this.author = source["author"];
	        this.date = source["date"];
	        this.msg = source["msg"];
	    }
	}
	export class FileDiff {
	    path: string;
	    status: string;
	    diff: string;
	    truncate: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FileDiff(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.status = source["status"];
	        this.diff = source["diff"];
	        this.truncate = source["truncate"];
	    }
	}
	export class CommitDiff {
	    hash: string;
	    files: FileDiff[];
	
	    static createFrom(source: any = {}) {
	        return new CommitDiff(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hash = source["hash"];
	        this.files = this.convertValues(source["files"], FileDiff);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace main {
	
	export class ImpactDTO {
	    changedModules: string[];
	    changedFiles: string[];
	    autoAddedModules: string[];
	    buildPlan: string[];
	
	    static createFrom(source: any = {}) {
	        return new ImpactDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.changedModules = source["changedModules"];
	        this.changedFiles = source["changedFiles"];
	        this.autoAddedModules = source["autoAddedModules"];
	        this.buildPlan = source["buildPlan"];
	    }
	}
	export class InitialStateDTO {
	    projectRoot: string;
	    config: config.Config;
	    hasGit: boolean;
	    hasPom: boolean;
	    hasMaven: boolean;
	    branches: string[];
	    currentBranch: string;
	    commits: git.Commit[];
	    moduleCount: number;
	    defaultOutputDir: string;
	
	    static createFrom(source: any = {}) {
	        return new InitialStateDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.projectRoot = source["projectRoot"];
	        this.config = this.convertValues(source["config"], config.Config);
	        this.hasGit = source["hasGit"];
	        this.hasPom = source["hasPom"];
	        this.hasMaven = source["hasMaven"];
	        this.branches = source["branches"];
	        this.currentBranch = source["currentBranch"];
	        this.commits = this.convertValues(source["commits"], git.Commit);
	        this.moduleCount = source["moduleCount"];
	        this.defaultOutputDir = source["defaultOutputDir"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class StartPackagingOpts {
	    commits: string[];
	    speedMode: string;
	    scopeMode: string;
	    smartDependency: boolean;
	    outputDir: string;
	
	    static createFrom(source: any = {}) {
	        return new StartPackagingOpts(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.commits = source["commits"];
	        this.speedMode = source["speedMode"];
	        this.scopeMode = source["scopeMode"];
	        this.smartDependency = source["smartDependency"];
	        this.outputDir = source["outputDir"];
	    }
	}
	export class SwitchBranchResult {
	    success: boolean;
	    message: string;
	    pullOutput: string;
	    branches: string[];
	    currentBranch: string;
	    commits: git.Commit[];
	
	    static createFrom(source: any = {}) {
	        return new SwitchBranchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.message = source["message"];
	        this.pullOutput = source["pullOutput"];
	        this.branches = source["branches"];
	        this.currentBranch = source["currentBranch"];
	        this.commits = this.convertValues(source["commits"], git.Commit);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace maven {
	
	export class GraphEdge {
	    from: string;
	    to: string;
	
	    static createFrom(source: any = {}) {
	        return new GraphEdge(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.from = source["from"];
	        this.to = source["to"];
	    }
	}
	export class GraphNode {
	    id: string;
	    artifactId: string;
	    groupId: string;
	    version: string;
	    packaging: string;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new GraphNode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.artifactId = source["artifactId"];
	        this.groupId = source["groupId"];
	        this.version = source["version"];
	        this.packaging = source["packaging"];
	        this.name = source["name"];
	    }
	}
	export class ModuleGraph {
	    nodes: GraphNode[];
	    edges: GraphEdge[];
	
	    static createFrom(source: any = {}) {
	        return new ModuleGraph(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nodes = this.convertValues(source["nodes"], GraphNode);
	        this.edges = this.convertValues(source["edges"], GraphEdge);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace timing {
	
	export class ModuleTiming {
	    module: string;
	    elapsedMs: number;
	    success: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ModuleTiming(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.module = source["module"];
	        this.elapsedMs = source["elapsedMs"];
	        this.success = source["success"];
	    }
	}
	export class Summary {
	    total: number;
	    modules: ModuleTiming[];
	    bottleneck: string;
	    bottleneckMs: number;
	
	    static createFrom(source: any = {}) {
	        return new Summary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.modules = this.convertValues(source["modules"], ModuleTiming);
	        this.bottleneck = source["bottleneck"];
	        this.bottleneckMs = source["bottleneckMs"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

