package main

import (
	"cloud.google.com/go/bigtable"
	"cloud.google.com/go/bigtable/bttest"
	"context"
	"errors"
	"flag"
	"fmt"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func main() {
	cfs := flag.String("cf", "", "Optional: the column families to create at startup. Format: a series of <instance>.<table>.<column familiy>, comma separated. Ex: \n          docker run -d spotify/bigtable-emulator -cf dev.records.data,dev.records.metadata")
	flag.Parse()

	srv, err := bttest.NewServer("0.0.0.0:9035")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error starting the server: %v\n", err)
		return
	}
	defer srv.Close()

	fmt.Printf("Trying to create the following column families: %s\n", *cfs)

	if err = createColumnFamilies(*cfs); err != nil {
		fmt.Fprintf(os.Stderr, "error creating the column familiies: %v\n", err)
		return
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	fmt.Printf("Bigtable emulator running on %s\n", srv.Addr)
	<-sigs
	fmt.Println("done")
}

func createColumnFamilies(specifications string) error {
	if specifications == "" {
		fmt.Printf("No specification found for family")
		return nil
	}

	ctx := context.Background()
	conn, err := grpc.Dial("localhost:9035", grpc.WithInsecure())
	if err != nil {
		return fmt.Errorf("could not connect to bigtable emulator: %v", err)
	}

	for _, specification := range strings.Split(specifications, ",") {
		specificationElements := strings.Split(specification, ".")
		if len(specificationElements) != 3 {
			return errors.New("format of column family to create is <instance>.<table>.<column family>")
		}

		instance := specificationElements[0]
		table := specificationElements[1]
		columnFamily := specificationElements[2]

		fmt.Printf("Creating Bigtable with table: %s, %s, %s\n", instance, table, columnFamily)

		client, err := bigtable.NewAdminClient(ctx, "dev", instance, option.WithGRPCConn(conn))
		if err != nil {
			return fmt.Errorf("failed to create admin client: %v", err)
		}

		tables, err := client.Tables(ctx)
		if err != nil {
			return fmt.Errorf("failed to fetch tables: %v", err)
		}

		if !tableExists(tables, table) {
			if err = client.CreateTable(ctx, table); err != nil {
				return err
			}
			fmt.Printf("Created Bigtable table: %s\n", table)
		}

		tableInfo, err := client.TableInfo(ctx, table)
		if err != nil {
			return fmt.Errorf("failed to fetch table info: %v", err)
		}

		if !columnFamilyExists(tableInfo.FamilyInfos, columnFamily) {
			fmt.Printf("creating %v.%v.%v column family\n", instance, table, columnFamily)
			if err := client.CreateColumnFamily(ctx, table, columnFamily); err != nil {
				return err
			}
		}
	}

	return nil
}

func tableExists(tables []string, table string) bool {
	for _, item := range tables {
		if item == table {
			return true
		}
	}
	return false
}

func columnFamilyExists(columnFamilies []bigtable.FamilyInfo, columnFamily string) bool {
	for _, family := range columnFamilies {
		if family.Name == columnFamily {
			return true
		}
	}
	return false
}
