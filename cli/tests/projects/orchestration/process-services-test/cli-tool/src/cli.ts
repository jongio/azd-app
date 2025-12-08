#!/usr/bin/env node
import { Command } from 'commander';

const program = new Command();

program
  .name('cli-tool')
  .description('Example CLI tool - process service with watch/build modes')
  .version('1.0.0');

program
  .command('greet <name>')
  .description('Greet someone')
  .action((name: string) => {
    console.log(`Hello, ${name}!`);
  });

program
  .command('echo <message>')
  .description('Echo a message')
  .action((message: string) => {
    console.log(message);
  });

program.parse();
